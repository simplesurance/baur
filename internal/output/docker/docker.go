package docker

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/config/types"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
)

const (
	// DefaultRegistry is the registry for that authentication data is used
	DefaultRegistry           = "https://index.docker.io/v1/"
	dockerRegistryDefaultPort = "5000"
)

// Client is a docker client
type Client struct {
	clt        *client.Client
	cfg        *configfile.ConfigFile
	debugLogFn func(string, ...any)
}

var defLogFn = func(string, ...any) {}

// NewClient initializes a new docker client.
// The following environment variables are respected:
// DOCKER_TLS_VERIFY
// DOCKER_CERT_PATH
// DOCKER_HOST to set the url to the docker server.
// DOCKER_API_VERSION to set the version of the API to reach, leave empty for latest.
// The Auth configuration is read from the user's config.json file for an
// Upload() operation, including credentials resolved dynamically via a
// configured credsStore/credHelpers (e.g. amazon-ecr-credential-helper),
// not just literal auths entries.
// The following files are checked in the order listed:
// - $DOCKER_CONFIG/config.json if DOCKER_CONFIG set in the environment,
// - $HOME/.docker/config.json
// - $HOME/.dockercfg
// If reading auth data from the config fails, a message is logged via the
// debugLogFn function but no error is returned. An Upload() operation would be
// done without authentication.
// DOCKER_AUTH_CONFIG is respected like by the docker CLI: if set, its auths
// entries (same format as in config.json, "auth" field required) are used
// before the ones from config.json. If it can not be parsed, docker's
// library prints an error to stderr and it is ignored.
func NewClient(debugLogFn func(string, ...any)) (*Client, error) {
	logFn := defLogFn
	if debugLogFn != nil {
		logFn = debugLogFn
	}

	dockerClt, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}

	cfg, err := config.Load("")
	if err != nil {
		debugLogFn("docker: reading auth data from user's config.json failed: %s", err)
		cfg = configfile.New("")
	}

	return &Client{
		clt:        dockerClt,
		cfg:        cfg,
		debugLogFn: logFn,
	}, nil
}

// getAuth returns authentication data for the given server, server can be a
// hostname or URL.
// if server is empty, the function panics.
// Credentials are resolved the same way the docker CLI does: via
// DOCKER_AUTH_CONFIG, the configured credsStore/credHelpers (e.g.
// amazon-ecr-credential-helper) or a literal auths entry in config.json.
// If that finds no credentials, the literal auths entries are also matched
// by hostname and by hostname with the default registry port (see
// literalAuth()).
// If no credentials are found, an AuthConfig with only ServerAddress set is
// returned.
// If a credential helper fails, the helper error is returned together with
// the matching literal auths entry, or with an AuthConfig with only
// ServerAddress set. The returned AuthConfig can always be used for a push,
// the error explains a failed push.
func (c *Client) getAuth(server string) (registry.AuthConfig, error) {
	if server == "" {
		panic("server is empty")
	}

	auth, credErr := c.cfg.GetAuthConfig(server)
	if credErr != nil {
		credErr = fmt.Errorf("resolving credentials for registry %q via credential helper failed: %w", server, credErr)
		c.debugLogFn("docker: %s, using auths entries from config.json", credErr)
	} else if auth != (types.AuthConfig{}) {
		c.debugLogFn("docker: using resolved credentials for registry %q", server)
		return toRegistryAuth(auth), nil
	}

	if auth, ok := c.literalAuth(server); ok {
		c.debugLogFn("docker: using auths entry from config.json for registry %q", server)
		// The daemon only uses the credentials if the host of
		// ServerAddress is the registry host, the key of the entry can
		// differ from it (e.g. no port).
		auth.ServerAddress = server
		return toRegistryAuth(auth), credErr
	}

	c.debugLogFn("docker: no auth configuration for registry %q found", server)
	return registry.AuthConfig{ServerAddress: server}, credErr
}

// literalAuth returns the literal auths entry of the config for server.
// An entry matches if its key is server, or if its key is a URL and:
//   - the URL has no port and its hostname, or its hostname with
//     dockerRegistryDefaultPort, is server,
//   - the URL has a port and its host:port is server, or is server with
//     dockerRegistryDefaultPort.
func (c *Client) literalAuth(server string) (types.AuthConfig, bool) {
	for reg, auth := range c.cfg.GetAuthConfigs() {
		if reg == server {
			return auth, true
		}

		u, err := url.Parse(reg)
		if err != nil || u.Hostname() == "" {
			continue
		}

		if u.Port() == "" {
			if u.Hostname() == server || net.JoinHostPort(u.Hostname(), dockerRegistryDefaultPort) == server {
				return auth, true
			}
			continue
		}

		registryHostPort := net.JoinHostPort(u.Hostname(), u.Port())
		if registryHostPort == server || registryHostPort == net.JoinHostPort(server, dockerRegistryDefaultPort) {
			return auth, true
		}
	}

	return types.AuthConfig{}, false
}

func toRegistryAuth(auth types.AuthConfig) registry.AuthConfig {
	return registry.AuthConfig{
		Username:      auth.Username,
		Password:      auth.Password,
		Auth:          auth.Auth,
		ServerAddress: auth.ServerAddress,
		IdentityToken: auth.IdentityToken,
		RegistryToken: auth.RegistryToken,
	}
}

// Upload tags and uploads an image to a docker registry repository.
// On success it returns the path  to the uploaded docker image, in the format:
// [registry]:/repository/tag
// If registryAddr is empty, the image is uploaded to Docker Hub with the
// credentials of DefaultRegistry and the path has no registry part.
func (c *Client) Upload(imageID, registryAddr, repository, tag string) (string, error) {
	addrRepo := registryAddr + "/" + repository
	authServer := registryAddr
	if registryAddr == "" {
		// DefaultRegistry is a URL, it is only valid as key of the auth
		// configuration, not in an image reference. A reference without
		// a registry part refers to Docker Hub.
		addrRepo = repository
		authServer = DefaultRegistry
	}
	destURI := addrRepo + ":" + tag

	c.debugLogFn("docker: creating tag, repo: %q, tag: %q referring to image %q", addrRepo, tag, imageID)
	err := c.clt.ImageTag(context.Background(), imageID, destURI)
	if err != nil {
		return "", fmt.Errorf("tagging image failed: %w", err)
	}

	auth, credErr := c.getAuth(authServer)
	authBytes, err := json.Marshal(auth)
	if err != nil {
		return "", fmt.Errorf("failed to marshal auth config: %w", err)
	}
	authBase64 := base64.URLEncoding.EncodeToString(authBytes)

	c.debugLogFn("docker: pushing image, name: %q, tag: %q", addrRepo, tag)
	resp, err := c.clt.ImagePush(context.Background(), destURI, image.PushOptions{
		RegistryAuth: authBase64,
	})
	if err != nil {
		return "", errors.Join(err, credErr)
	}
	defer resp.Close()

	if err := pushStreamError(resp, c.debugLogFn); err != nil {
		return "", errors.Join(err, credErr)
	}

	return destURI, nil
}

// pushStreamError reads Docker's push progress stream until EOF and logs
// every line via logFn.
// Docker's push API reports registry-level failures (auth denied,
// permission errors, etc.) as an "error" field inside this JSON stream,
// not as a Go error from ImagePush() - that only fails on transport-level
// problems. Without checking the stream, a denied push is silently treated
// as a success.
// It returns the last error reported in the stream, or an error if reading
// the stream failed.
func pushStreamError(stream io.Reader, logFn func(string, ...any)) error {
	var pushErr error

	r := bufio.NewReader(stream)
	for {
		line, err := r.ReadString('\n')
		// The last line can end without '\n', it is then returned
		// together with io.EOF.
		if line != "" {
			logFn("docker: %s", strings.TrimRight(line, "\r\n"))

			if lineErr := pushStreamLineError(line); lineErr != nil {
				pushErr = lineErr
			}
		}

		if errors.Is(err, io.EOF) {
			return pushErr
		}
		if err != nil {
			return fmt.Errorf("reading push response failed: %w", err)
		}
	}
}

// pushStreamLineError returns a non-nil error if line, a single line of
// Docker's push progress stream, is a JSON message reporting a
// registry-level failure. It returns nil for progress messages and for
// lines that aren't valid JSON.
func pushStreamLineError(line string) error {
	var msg struct {
		Error       string `json:"error"`
		ErrorDetail struct {
			Message string `json:"message"`
		} `json:"errorDetail"`
	}

	if err := json.Unmarshal([]byte(line), &msg); err != nil {
		return nil
	}

	// "error" is deprecated in the Docker API and newer API types only
	// define "errorDetail", prefer it and fall back to "error".
	errMsg := msg.ErrorDetail.Message
	if errMsg == "" {
		errMsg = msg.Error
	}
	if errMsg == "" {
		return nil
	}

	return fmt.Errorf("push failed: %s", errMsg)
}

// SizeBytes returns the size of an image in Bytes.
func (c *Client) SizeBytes(imageID string) (int64, error) {
	img, _, err := c.clt.ImageInspectWithRaw(context.Background(), imageID) //nolint: staticcheck
	if err != nil {
		return -1, err
	}

	return img.Size, nil
}

// Exists return true if the image with the given ID exist, otherwise false.
func (c *Client) Exists(imageID string) (bool, error) {
	_, _, err := c.clt.ImageInspectWithRaw(context.Background(), imageID) //nolint: staticcheck
	if err != nil {
		if cerrdefs.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
