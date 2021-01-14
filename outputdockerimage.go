package baur

import (
	"fmt"

	"github.com/simplesurance/baur/v1/internal/digest"
	"github.com/simplesurance/baur/v1/internal/fs"
)

type DockerInfoClient interface {
	Size(imageID string) (int64, error)
	Exists(imageID string) (bool, error)
}

// OutputDockerImage is a docker image artifact.
type OutputDockerImage struct {
	name               string
	ImageID            string
	UploadDestinations []*UploadInfoDocker
	dockerClient       DockerInfoClient
	digest             *digest.Digest
}

func NewOutputDockerImageFromIIDFile(
	dockerClient DockerInfoClient,
	name,
	iidfile string,
	uploadDest []*UploadInfoDocker,
) (*OutputDockerImage, error) {
	id, err := fs.FileReadLine(iidfile)
	if err != nil {
		return nil, fmt.Errorf("reading %s failed: %w", iidfile, err)
	}

	digest, err := digest.FromString(id)
	if err != nil {
		return nil, fmt.Errorf("image id %q read from %q has an invalid format: %w", id, iidfile, err)
	}

	return &OutputDockerImage{
		name:               name,
		dockerClient:       dockerClient,
		ImageID:            id,
		UploadDestinations: uploadDest,
		digest:             digest,
	}, nil
}

func (d *OutputDockerImage) String() string {
	return fmt.Sprintf("docker image: %s", d.ImageID)
}

func (d *OutputDockerImage) Name() string {
	return d.name
}

func (d *OutputDockerImage) Type() OutputType {
	return DockerOutput
}

func (d *OutputDockerImage) Exists() (bool, error) {
	return d.dockerClient.Exists(d.ImageID)
}

// Digest returns the imageID as a digest object. The method always returns a nil error.
func (d *OutputDockerImage) Digest() (*digest.Digest, error) {
	return d.digest, nil
}

func (d *OutputDockerImage) Size() (uint64, error) {
	size, err := d.dockerClient.Size(d.ImageID)
	if err != nil {
		return 0, nil
	}

	return uint64(size), nil
}
