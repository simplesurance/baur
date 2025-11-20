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
	"net/url"

	"github.com/docker/cli/cli/config"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
)

// DefaultRegistry is the registry for that authentication data is used
const (
	DefaultRegistry           = "https://index.docker.io/v1/"
	dockerRegistryDefaultPort = "5000"
)

// Client is a docker client
type Client struct {
	clt        *client.Client
	auths      map[string]registry.AuthConfig
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
// Upload() operation.
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

	auths := make(map[string]registry.AuthConfig)
	cfg, err := config.Load("")
	if err != nil {
		debugLogFn("docker: reading auth data from user's config.json failed: %s", err)
	} else {
		for reg, auth := range cfg.AuthConfigs {
			auths[reg] = registry.AuthConfig{
				Username:      auth.Username,
				Password:      auth.Password,
				Auth:          auth.Auth,
				ServerAddress: auth.ServerAddress,
				IdentityToken: auth.IdentityToken,
				RegistryToken: auth.RegistryToken,
			}
		}
	}

	return &Client{
		clt:        dockerClt,
		auths:      auths,
		debugLogFn: logFn,
	}, nil
}

func hostPort(host, port string) string {
	return fmt.Sprintf("%s:%s", host, port)
}

// getAuth returns authentication date for the given server, server can be a
// hostname or URL.
// if server is empty, the function panics.
// if the client has no authentication data (c.auths is empty), an empty
// AuthConfiguration is returned.
// If server is not empty, authentication data is returned for a registry with
// a matching address.
func (c *Client) getAuth(server string) registry.AuthConfig {
	if len(c.auths) == 0 {
		return registry.AuthConfig{}
	}

	if server == "" {
		panic("server is empty")
	}

	if len(server) == 0 {
		for registry, auth := range c.auths {
			if registry == DefaultRegistry {
				c.debugLogFn("docker: no registry specified, using auth data for default registry %q", registry)
				return auth
			}
		}
	}

	for reg, cfg := range c.auths {
		c.debugLogFn("docker: found credentials for registry %q, searching %q credentials", reg, server)

		if reg == server {
			c.debugLogFn("docker: using auth data for registry '%s'", reg)
			return cfg
		}

		url, err := url.Parse(reg)
		if err != nil || url.Hostname() == "" {
			continue
		}

		if url.Port() == "" {
			if url.Hostname() == server || hostPort(url.Hostname(), dockerRegistryDefaultPort) == server {
				c.debugLogFn("docker: using auth data for registry '%s'", reg)
				return cfg
			}

			continue
		}

		registryHostPort := hostPort(url.Hostname(), url.Port())
		if registryHostPort == server || registryHostPort == hostPort(server, dockerRegistryDefaultPort) {
			c.debugLogFn("docker: using auth data for registry '%s'", reg)
			return cfg
		}
	}

	c.debugLogFn("docker: no auth configuration for registry %q found", server)

	return registry.AuthConfig{ServerAddress: server}
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

	auth := c.getAuth(registryAddr)
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

	for {
		outStream.Flush()
		line, err := outBuf.ReadString('\n')
		if errors.Is(err, io.EOF) {
			break
		}

		c.debugLogFn("docker: " + line)
	}

	if err != nil {
		return "", err
	}

	return destURI, nil
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
		if client.IsErrNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
