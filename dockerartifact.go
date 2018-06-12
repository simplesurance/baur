package baur

import (
	"errors"
	"fmt"

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

// String returns the string representation of the artifact
func (d *DockerArtifact) String() string {
	return d.ImageIDFile
}

// LocalPath returns the local path to the artifact
func (d *DockerArtifact) LocalPath() string {
	return d.ImageIDFile
}

// UploadDestination returns the upload destination
func (d *DockerArtifact) UploadDestination() string {
	return fmt.Sprintf("docker: %s:%s", d.Repository, d.Tag)
}
