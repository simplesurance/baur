package baur

import (
	"fmt"
	"path/filepath"

	"github.com/simplesurance/baur/digest"
)

type OutputType int

const (
	DockerOutput OutputType = iota
	FileOutput
)

func (o OutputType) String() string {
	switch o {
	case DockerOutput:
		return "docker"
	case FileOutput:
		return "file"

	default:
		return "invalid OutputType"
	}
}

type Output interface {
	Name() string
	String() string
	Exists() (bool, error)
	Size() (uint64, error)
	Digest() (*digest.Digest, error)
	Type() OutputType
}

func OutputsFromTask(dockerClient DockerInfoClient, task *Task) ([]Output, error) {
	var result []Output

	// TODO: move each loop to an own sub-function

	for _, dockerOutput := range task.Outputs.DockerImage {
		d, err := NewOutputDockerImageFromIIDFile(
			dockerClient,
			dockerOutput.IDFile,
			filepath.Join(task.Directory, dockerOutput.IDFile),
			&UploadInfoDocker{
				Registry:   dockerOutput.RegistryUpload.Registry,
				Repository: dockerOutput.RegistryUpload.Repository,
				Tag:        dockerOutput.RegistryUpload.Tag,
			},
		)

		if err != nil {
			return nil, err
		}

		result = append(result, d)
	}

	for _, fileOutput := range task.Outputs.File {
		var s3Upload *UploadInfoS3
		var fileCopyUpload *UploadInfoFileCopy

		if fileOutput.S3Upload.IsEmpty() && fileOutput.FileCopy.IsEmpty() {
			return nil, fmt.Errorf("no upload method for output %q is specified", fileOutput.Path)
		}

		// TODO: use pointers in the outputfile struct for filecopy and S3 instead of having to provide and use IsEmpty)

		if !fileOutput.S3Upload.IsEmpty() {
			s3Upload = &UploadInfoS3{
				Bucket: fileOutput.S3Upload.Bucket,
				Key:    fileOutput.S3Upload.DestFile,
			}
		}

		if !fileOutput.FileCopy.IsEmpty() {
			fileCopyUpload = &UploadInfoFileCopy{DestinationPath: fileOutput.FileCopy.Path}
		}

		result = append(result, NewOutputFile(
			fileOutput.Path,
			filepath.Join(task.Directory, fileOutput.Path),
			s3Upload,
			fileCopyUpload,
		))
	}

	return result, nil
}
