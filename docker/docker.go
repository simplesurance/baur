package docker

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"docker.io/go-docker"
	"docker.io/go-docker/api/types"
	"github.com/pkg/errors"
)

// Client is a docker client
type Client struct {
	clt      *docker.Client
	authData string
}

func base64AuthData(user, password string) (string, error) {
	ac := types.AuthConfig{
		Username: user,
		Password: password,
	}

	js, err := json.Marshal(ac)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(js), nil
}

// NewClientwAuth intializes a new docker client with the given docker registry
// authentication data.
// The following environment variables are respected:
// Use DOCKER_HOST to set the url to the docker server.
// Use DOCKER_API_VERSION to set the version of the API to reach, leave empty for latest.
// Use DOCKER_CERT_PATH to load the TLS certificates from.
// Use DOCKER_TLS_VERIFY to enable or disable TLS verification, off by default.
func NewClientwAuth(user, password string) (*Client, error) {
	clt, err := docker.NewEnvClient()
	if err != nil {
		return nil, err
	}

	authData, err := base64AuthData(user, password)
	if err != nil {
		return nil, err
	}

	return &Client{
		clt:      clt,
		authData: authData,
	}, nil
}

// NewClient returns a new docker client
func NewClient() (*Client, error) {
	clt, err := docker.NewEnvClient()
	if err != nil {
		return nil, err
	}

	return &Client{
		clt: clt,
	}, nil
}

func serverRespIsErr(in []byte) error {
	var m map[string]interface{}
	err := json.Unmarshal(in, &m)
	if err != nil {
		return err
	}

	if _, exist := m["error"]; exist {
		prettyErr, err := json.MarshalIndent(&m, "", "  ")
		if err != nil {
			return errors.New(fmt.Sprint(m))
		}

		return errors.New(string(prettyErr))
	}

	return nil
}

// Upload tags and uploads an image into a docker registry repository
func (c *Client) Upload(ctx context.Context, image, dest string) (string, error) {
	err := c.clt.ImageTag(ctx, image, dest)
	if err != nil {
		return "", errors.Wrapf(err, "tagging image failed")
	}

	closer, err := c.clt.ImagePush(ctx, dest, types.ImagePushOptions{
		RegistryAuth: c.authData,
	})
	if err != nil {
		return "", errors.Wrapf(err, "pushing image failed")
	}

	defer closer.Close()

	r := bufio.NewReader(closer)
	for {
		status, err := r.ReadBytes('\n')
		if err == io.EOF {
			break
		}

		if err := serverRespIsErr(status); err != nil {
			return "", errors.Wrapf(err, "pushing image failed")
		}
	}

	return dest, nil
}

// Size returns the size of an image in Bytes
func (c *Client) Size(ctx context.Context, imageID string) (int64, error) {
	summaries, err := c.clt.ImageList(ctx, types.ImageListOptions{})
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

//ImageDigest retrieves the image digest from a remote registry and returns it
func (c *Client) ImageDigest(ctx context.Context, image string) (string, error) {
	ins, err := c.clt.DistributionInspect(ctx, image, c.authData)
	if err != nil {
		return "", errors.Wrap(err, "retrieving image information failed")
	}

	return ins.Descriptor.Digest.String(), nil
}
