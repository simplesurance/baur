package docker

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/url"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/pkg/errors"
)

// DefaultRegistry is the registry for that authentication data is used
const DefaultRegistry = "https://index.docker.io/v1/"
const dockerRegistryDefaultPort = "5000"

// Client is a docker client
type Client struct {
	clt        *docker.Client
	auths      *docker.AuthConfigurations
	auth       *docker.AuthConfiguration
	debugLogFn func(string, ...interface{})
}

var defLogFn = func(string, ...interface{}) {}

// NewClientwAuth intializes a new docker client.
// The username and password is used to authenticate at the registry for an
// Upload() = docker push) operation.
// The following environment variables are respected:
// Use DOCKER_HOST to set the url to the docker server.
// Use DOCKER_API_VERSION to set the version of the API to reach, leave empty for latest.
// Use DOCKER_CERT_PATH to load the TLS certificates from.
// Use DOCKER_TLS_VERIFY to enable or disable TLS verification, off by default.
func NewClientwAuth(debugLogFn func(string, ...interface{}), username, password string) (*Client, error) {
	logFn := defLogFn
	if debugLogFn != nil {
		logFn = debugLogFn
	}

	dockerClt, err := docker.NewClientFromEnv()
	if err != nil {
		return nil, err
	}

	return &Client{
		clt: dockerClt,
		auth: &docker.AuthConfiguration{
			Username: username,
			Password: password,
		},
		debugLogFn: logFn,
	}, nil
}

// NewClient initializes a new docker client.
// It supports the same environment variables then NewClientwAuth().
// The Auth configuration is read from the user's config.json file for an
// Upload() operation.
// The following files are checked in the order listed:
// - $DOCKER_CONFIG/config.json if DOCKER_CONFIG set in the environment,
// - $HOME/.docker/config.json
// - $HOME/.dockercfg
// If reading auth data from the config fails, a message is logged via the
// debugLogFn function but no error is returned. An Upload() operation would be
// done without authentication.
func NewClient(debugLogFn func(string, ...interface{})) (*Client, error) {
	logFn := defLogFn
	if debugLogFn != nil {
		logFn = debugLogFn
	}

	dockerClt, err := docker.NewClientFromEnv()
	if err != nil {
		return nil, err
	}

	auths, err := docker.NewAuthConfigurationsFromDockerCfg()
	if err != nil {
		debugLogFn("docker: reading auth data from user's config.json failed: %s", err)
		auths = nil
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
// getAuth returns c.auth if it's not nil otherwise:
// if server is empty, the function panics.
// If server is not empty, authentication data is returned for a registry with a matching address.
// if the client has no authentication data or no authentication data for the
// server exist, an AuthConfiguration is returned with only the
// ServerAddress set to server.
func (c *Client) getAuth(server string) docker.AuthConfiguration {
	if c.auth != nil {
		return *c.auth
	}

	if c.auths == nil || len(c.auths.Configs) == 0 {
		return docker.AuthConfiguration{}
	}

	if server == "" {
		panic("server is empty")
	}

	if len(server) == 0 {
		for registry, auth := range c.auths.Configs {
			if registry == DefaultRegistry {
				c.debugLogFn("docker: no registry specified, using auth data for default registry %q", registry)
				return auth
			}
		}
	}

	for _, cfg := range c.auths.Configs {
		c.debugLogFn("docker: found credentials for registry %q, searching %q credentials", cfg.ServerAddress, server)

		if cfg.ServerAddress == server {
			c.debugLogFn("docker: using auth data for registry '%s'", cfg.ServerAddress)
			return cfg
		}

		url, err := url.Parse(cfg.ServerAddress)
		if err != nil || url.Hostname() == "" {
			continue
		}

		if url.Port() == "" {
			if url.Hostname() == server || hostPort(url.Hostname(), dockerRegistryDefaultPort) == server {
				c.debugLogFn("docker: using auth data for registry '%s'", cfg.ServerAddress)
				return cfg
			}

			continue
		}

		registryHostPort := hostPort(url.Hostname(), url.Port())
		if registryHostPort == server || registryHostPort == hostPort(server, dockerRegistryDefaultPort) {
			c.debugLogFn("docker: using auth data for registry '%s'", cfg.ServerAddress)
			return cfg
		}
	}

	c.debugLogFn("docker: no auth configuration for registry %q found", server)

	return docker.AuthConfiguration{ServerAddress: server}
}

// Upload tags and uploads an image into a docker registry repository
// destURI format: <repository>:<tag>
func (c *Client) Upload(image, registryAddr, repository, tag string) (string, error) {
	var addrRepo string
	var destURI string

	if registryAddr == "" {
		registryAddr = DefaultRegistry
	}

	addrRepo = registryAddr + "/" + repository
	destURI = registryAddr + "/" + repository + ":" + tag

	c.debugLogFn("docker: creating tag, repo: %q, tag: %q referring to image %q", addrRepo, tag, image)
	err := c.clt.TagImage(image, docker.TagImageOptions{
		Repo: addrRepo,
		Tag:  tag,
	})
	if err != nil {
		return "", errors.Wrapf(err, "tagging image failed")
	}

	auth := c.getAuth(registryAddr)

	var outBuf bytes.Buffer
	outStream := bufio.NewWriter(&outBuf)

	c.debugLogFn("docker: pushing image, name: %q, tag: %q", addrRepo, tag)
	err = c.clt.PushImage(docker.PushImageOptions{
		Name:         addrRepo,
		Tag:          tag,
		OutputStream: outStream,
	}, auth)

	for {
		outStream.Flush()
		line, err := outBuf.ReadString('\n')
		if err == io.EOF {
			break
		}

		c.debugLogFn("docker: " + line)
	}

	if err != nil {
		return "", err
	}

	return destURI, nil
}

// Size returns the size of an image in Bytes.
func (c *Client) Size(imageID string) (int64, error) {
	img, err := c.clt.InspectImage(imageID)
	if err != nil {
		return -1, err
	}

	return img.VirtualSize, nil
}
