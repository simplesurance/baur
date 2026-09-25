package docker

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/config/types"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
)

// DefaultRegistry is the registry for that authentication data is used
const DefaultRegistry = "https://index.docker.io/v1/"

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
// Credentials are resolved the same way the docker CLI does: a literal
// auths entry in config.json if present, otherwise dynamically via the
// configured credsStore/credHelpers (e.g. amazon-ecr-credential-helper).
// If no credentials are found, an AuthConfig with only ServerAddress set is
// returned.
// An error is returned if resolving the credentials failed, e.g. because a
// configured credential helper failed.
func (c *Client) getAuth(server string) (registry.AuthConfig, error) {
	if server == "" {
		panic("server is empty")
	}

	auth, err := c.cfg.GetAuthConfig(server)
	if err != nil {
		return registry.AuthConfig{}, fmt.Errorf("resolving credentials for registry %q failed: %w", server, err)
	}

	if auth == (types.AuthConfig{}) {
		c.debugLogFn("docker: no auth configuration for registry %q found", server)
		return registry.AuthConfig{ServerAddress: server}, nil
	}

	c.debugLogFn("docker: using resolved credentials for registry %q", server)
	return registry.AuthConfig{
		Username:      auth.Username,
		Password:      auth.Password,
		Auth:          auth.Auth,
		ServerAddress: auth.ServerAddress,
		IdentityToken: auth.IdentityToken,
		RegistryToken: auth.RegistryToken,
	}, nil
}

// Upload tags and uploads an image to a docker registry repository.
// On success it returns the path  to the uploaded docker image, in the format:
// [registry]:/repository/tag
func (c *Client) Upload(imageID, registryAddr, repository, tag string) (string, error) {
	var addrRepo string
	var destURI string

	if registryAddr == "" {
		registryAddr = DefaultRegistry
	}

	addrRepo = registryAddr + "/" + repository
	destURI = registryAddr + "/" + repository + ":" + tag

	c.debugLogFn("docker: creating tag, repo: %q, tag: %q referring to image %q", addrRepo, tag, imageID)
	err := c.clt.ImageTag(context.Background(), imageID, addrRepo+":"+tag)
	if err != nil {
		return "", fmt.Errorf("tagging image failed: %w", err)
	}

	auth, err := c.getAuth(registryAddr)
	if err != nil {
		return "", err
	}
	authBytes, err := json.Marshal(auth)
	if err != nil {
		return "", fmt.Errorf("failed to marshal auth config: %w", err)
	}
	authBase64 := base64.URLEncoding.EncodeToString(authBytes)

	var outBuf bytes.Buffer
	outStream := bufio.NewWriter(&outBuf)

	c.debugLogFn("docker: pushing image, name: %q, tag: %q", addrRepo, tag)
	resp, err := c.clt.ImagePush(context.Background(), addrRepo+":"+tag, image.PushOptions{
		RegistryAuth: authBase64,
	})
	if err != nil {
		return "", err
	}
	defer resp.Close()

	_, err = io.Copy(outStream, resp)
	if err != nil {
		return "", err
	}

	// Docker's push API reports registry-level failures (auth denied,
	// permission errors, etc.) as an "error" field inside this JSON stream,
	// not as a Go error from ImagePush()/io.Copy() above - both of those
	// only fail on transport-level problems. Without checking for it here,
	// a denied push is silently treated as a success.
	var pushErr error
	for {
		outStream.Flush()
		line, err := outBuf.ReadString('\n')
		if errors.Is(err, io.EOF) {
			break
		}

		c.debugLogFn("docker: " + line)

		if lineErr := pushStreamLineError(line); lineErr != nil {
			pushErr = lineErr
		}
	}

	if pushErr != nil {
		return "", pushErr
	}

	return destURI, nil
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
