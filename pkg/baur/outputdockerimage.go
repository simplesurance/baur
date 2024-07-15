package baur

import (
	"fmt"

	"github.com/simplesurance/baur/v5/internal/digest"
	"github.com/simplesurance/baur/v5/internal/fs"
)

// DockerInfoClient is an interface fo retrieving information about a docker image.
type DockerInfoClient interface {
	// Size returns the size of an image in bytes
	SizeBytes(imageID string) (int64, error)
	// Exists returns true and no error if an image with the image ID imageID exists locally.
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

// NewOutputDockerImageFromIIDFile instantiates a new OutputDockerImage.
// name only acts as an identifier for the image.
// iidfile is the path to a file containing the ID of the docker image.
// uploadDest describes where the image should be uploaded to.
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

func (d *OutputDockerImage) SizeBytes() (uint64, error) {
	size, err := d.dockerClient.SizeBytes(d.ImageID)
	if err != nil {
		return 0, nil
	}

	return uint64(size), nil
}
