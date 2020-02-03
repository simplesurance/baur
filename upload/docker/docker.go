package docker

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/pkg/errors"
)

// DefaultRegistry is the registry for that authentication data is used
const DefaultRegistry = "https://index.docker.io/v1/"

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

// getAuth returns c.auth if it's not nil otherwise the matching authentication
// data for the server from docker's config.json
// if server is empty the first found entry is returned
// We do not have to care about name resolution, the docker CLI tools also
// don't do it, if you want to addressa server via and via IP you have to do
// "docker login" with each and the passed address ends up in the config.json
func (c *Client) getAuth(server string) docker.AuthConfiguration {
	if c.auth != nil {
		return *c.auth
	}

	if c.auths == nil || len(c.auths.Configs) == 0 {
		return docker.AuthConfiguration{}
	}

	if len(server) != 0 {
		for registry, v := range c.auths.Configs {
			url, err := url.Parse(registry)
			if err != nil {
				c.debugLogFn("docker: could not parse registry URL '%s' from docker auth config: %s", registry, err)
				continue
			}

			registryAddr := url.Host
			if url.Port() != "" {
				registryAddr += ":" + url.Port()
			}

			if registryAddr == server {
				c.debugLogFn("docker: using auth data for registry '%s'", registryAddr)
				return v
			}
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

// parseRepositoryURI splits a URI in the format:
// [<host[:port]>]/<owner>/<repository>:<tag> into it's parts
func parseRepositoryURI(dest string) (server, repository, tag string, err error) {
	spl := strings.SplitN(dest, "/", 3)
	if len(spl) == 3 {
		server = spl[0]
		repository = spl[1] + "/" + spl[2]
	} else if len(spl) == 2 {
		repository = spl[0] + "/" + spl[1]
	} else {
		return "", "", "", errors.New("invalid repository URI")
	}

	spl = strings.Split(repository, ":")
	// can contain up to 2 colons, one for the port in the server address,
	// one for the tag
	if len(spl) < 2 {
		return "", "", "", errors.New("parsing tag failed")
	}

	tag = spl[len(spl)-1]
	repository = spl[len(spl)-2]

	return
}

// Upload tags and uploads an image into a docker registry repository
// destURI format: [<server[:port]>/]<owner>/<repository>:<tag>
func (c *Client) Upload(image, destURI string) (string, error) {
	server, repository, tag, err := parseRepositoryURI(destURI)
	if err != nil {
		return "", err
	}

	err = c.clt.TagImage(image, docker.TagImageOptions{
		Repo: repository,
		Tag:  tag,
	})
	if err != nil {
		return "", errors.Wrapf(err, "tagging image failed")
	}

	auth := c.getAuth(server)

	var outBuf bytes.Buffer
	outStream := bufio.NewWriter(&outBuf)

	err = c.clt.PushImage(docker.PushImageOptions{
        Registry:   server,
		Name:         repository,
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

// Size returns the size of an image in Bytes
func (c *Client) Size(imageID string) (int64, error) {
	summaries, err := c.clt.ListImages(docker.ListImagesOptions{})
	if err != nil {
		return -1, errors.Wrap(err, "fetching imagelist failed")
	}

	for _, sum := range summaries {
		if sum.ID == imageID {
			if sum.VirtualSize <= 0 {
				return -1, fmt.Errorf("docker returned invalid image size %q", sum.VirtualSize)
			}
			return sum.VirtualSize, nil
		}
	}

	return -1, os.ErrNotExist
}
