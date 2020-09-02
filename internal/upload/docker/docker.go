package docker

import (
	"bufio"
	"bytes"
	"io"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/pkg/errors"
)

// DefaultRegistry is the registry for that authentication data is used
const DefaultRegistry = "https://index.docker.io/v1/"

// Client is a docker client
type Client struct {
	clt        *docker.Client
	auths      *docker.AuthConfigurations
	debugLogFn func(string, ...interface{})
}

var defLogFn = func(string, ...interface{}) {}

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

// getAuth returns if server is not empty and auth data for the server exist, the authentication data of it.
// If server is an empty string, the authentication data for DefaultRegistry is returned if it exists.
func (c *Client) getAuth(server string) docker.AuthConfiguration {
	if c.auths == nil || len(c.auths.Configs) == 0 {
		return docker.AuthConfiguration{}
	}

	if len(server) != 0 {
		for registry, v := range c.auths.Configs {
			if registry == server {
				c.debugLogFn("docker: using auth data for registry '%s'", registry)
				return v
			}

			c.debugLogFn("docker: found credentials for registry %q, searching %q credentials", registry, server)
		}
	}

	// try to find an entry for DefaultRegistry
	for registry, auth := range c.auths.Configs {
		if registry == DefaultRegistry {
			c.debugLogFn("docker: using auth data for default registry '%s'", registry)
			return auth
		}
	}

	// otherwise use the first entry in the map
	for registry, auth := range c.auths.Configs {
		c.debugLogFn("docker: using auth data for registry '%s'", registry)

		return auth
	}

	// this can't happen, auths must have been non-empty and
	// therefore we can return the first element
	panic("docker: could not find any auth data in a config.json")
}

// Upload tags and uploads an image to a docker registry repository.
// On success it returns the path  to the uploaded docker image, in the format:
// [registry]:/repository/tag
func (c *Client) Upload(image, registryAddr, repository, tag string) (string, error) {
	var addrRepo string
	var destURI string

	if registryAddr == "" {
		addrRepo = repository
		destURI = repository + ":" + tag
	} else {
		addrRepo = registryAddr + "/" + repository
		destURI = registryAddr + "/" + repository + ":" + tag
	}

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

// Exists return true if the image with the given ID exist, otherwise false.
func (c *Client) Exists(imageID string) (bool, error) {
	_, err := c.clt.InspectImage(imageID)
	if err != nil && err != docker.ErrNoSuchImage {
		return false, err
	}

	return err != docker.ErrNoSuchImage, nil
}
