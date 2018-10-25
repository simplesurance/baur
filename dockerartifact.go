package baur

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/simplesurance/baur/digest"
	"github.com/simplesurance/baur/fs"
	"github.com/simplesurance/baur/upload"
)

// DockerArtifact is a docker container artifact
type DockerArtifact struct {
	ImageIDFile string
	Tag         string
	Repository  string
}

// Exists returns true if the ImageIDFile exists
func (d *DockerArtifact) Exists() bool {
	return fs.FileExists(d.ImageIDFile)
}

// ImageID reads the image from ImageIDFile
func (d *DockerArtifact) ImageID() (string, error) {
	id, err := fs.FileReadLine(d.ImageIDFile)
	if err != nil {
		return "", err
	}

	if len(id) == 0 {
		return "", errors.New("file is empty")
	}

	return id, nil
}

// UploadJob returns a upload.DockerJob for the artifact
func (d *DockerArtifact) UploadJob() (upload.Job, error) {
	id, err := d.ImageID()
	if err != nil {
		return nil, err
	}

	return &upload.DockerJob{
		ImageID:    id,
		Repository: d.Repository,
		Tag:        d.Tag,
	}, nil
}

// String returns the absolute path to the ImageID file
func (d *DockerArtifact) String() string {
	return d.LocalPath()
}

// LocalPath returns the local path to the artifact
func (d *DockerArtifact) LocalPath() string {
	return d.ImageIDFile
}

// Name returns the docker repository name
func (d *DockerArtifact) Name() string {
	return d.Repository
}

// UploadDestination returns the upload destination
func (d *DockerArtifact) UploadDestination() string {
	return fmt.Sprintf("%s:%s", d.Repository, d.Tag)
}

// Digest returns the image ID as Digest object
func (d *DockerArtifact) Digest() (*digest.Digest, error) {
	id, err := d.ImageID()
	if err != nil {
		return nil, errors.Wrap(err, "reading imageID from file failed")
	}

	digest, err := digest.FromString(id)
	if err != nil {
		return nil, errors.Wrap(err, "converting imageID to digest failed")
	}

	return digest, nil
}

// Size returns the size of the docker image in bytes
func (d *DockerArtifact) Size(b *BuildOutputBackends) (int64, error) {
	id, err := d.ImageID()
	if err != nil {
		return -1, errors.Wrap(err, "reading imageID from file failed")
	}

	return b.DockerClt.Size(context.Background(), id)
}
