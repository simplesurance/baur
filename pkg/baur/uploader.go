package baur

import (
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"time"
)

// S3Uploader is an interface for uploading files to AWS S3 buckets.
type S3Uploader interface {
	Upload(filepath, bucket, key string) (string, error)
}

// DockerImgUploader is an interface for uploading docker images to a docker registry.
type DockerImgUploader interface {
	Upload(image, registryAddr, repository, tag string) (string, error)
}

// FileCopyUploader is an interface for copying files from one directory to another.
type FileCopyUploader interface {
	Upload(src string, dst string) (string, error)
}

// Uploader uploads outputs, produced by task run, to remote locations.
type Uploader struct {
	dockerclient     DockerImgUploader
	s3client         S3Uploader
	filecopyUploader FileCopyUploader
}

func NewUploader(dockerClient DockerImgUploader, s3client S3Uploader, filecopyUploader FileCopyUploader) *Uploader {
	return &Uploader{
		dockerclient:     dockerClient,
		s3client:         s3client,
		filecopyUploader: filecopyUploader,
	}
}

// UploadResult is the result of an upload operation.
type UploadResult struct {
	// Output is the output that was uploaded.
	Output Output
	URL    string
	Start  time.Time
	Stop   time.Time
	Method UploadMethod
}

// UploadStartFn is a function that is called before the upload operation starts.
type UploadStartFn func(Output, UploadInfo)

// UploadResultFn is a function that is called after an upload finishes.
type UploadResultFn func(Output, *UploadResult)

// Upload uploads an output to remote locations.
// Output must be a *OutputDockerImage or *OutputFile type storing or more upload locations.
// Immediately before the upload starts uploadStartCb is called, when the
// upload finished resultCb is called.
func (u *Uploader) Upload(output Output, uploadStartCb UploadStartFn, resultCb UploadResultFn) error {
	switch o := output.(type) {
	case *OutputDockerImage:
		if o.UploadDestinations == nil {
			return errors.New("uploadDestination is nil")
		}

		for _, dest := range o.UploadDestinations {
			uploadStartCb(o, dest)

			result, err := u.dockerImage(o, dest)
			if err != nil {
				return fmt.Errorf("docker upload failed: %w", err)
			}

			resultCb(o, result)
		}

	case *OutputFile:
		for _, dest := range o.UploadsFilecopy {
			uploadStartCb(o, dest)

			result, err := u.fileCopy(o, dest)
			if err != nil {
				return fmt.Errorf("filecopy failed: %w", err)
			}

			resultCb(o, result)
		}

		for _, dest := range o.UploadsS3 {
			uploadStartCb(o, dest)

			result, err := u.s3(o, dest)
			if err != nil {
				return fmt.Errorf("s3 upload failed: %w", err)
			}

			resultCb(o, result)
		}

	default:
		return fmt.Errorf("unsupported output type: %s", reflect.TypeOf(output).Kind())
	}

	return nil
}

func (u *Uploader) dockerImage(o *OutputDockerImage, dest *UploadInfoDocker) (*UploadResult, error) {
	startTime := time.Now()

	url, err := u.dockerclient.Upload(
		o.ImageID,
		dest.Registry,
		dest.Repository,
		dest.Tag,
	)
	if err != nil {
		return nil, err
	}

	return &UploadResult{
		Start:  startTime,
		Stop:   time.Now(),
		Method: UploadMethodDocker,
		Output: o,
		URL:    url,
	}, nil
}

func (u *Uploader) fileCopy(o *OutputFile, dest *UploadInfoFileCopy) (*UploadResult, error) {
	startTime := time.Now()

	destFile := filepath.Join(dest.Path, filepath.Base(o.absPath))

	url, err := u.filecopyUploader.Upload(o.absPath, destFile)
	if err != nil {
		return nil, err
	}

	return &UploadResult{
		Start:  startTime,
		Stop:   time.Now(),
		Method: UploadMethodFilecopy,
		Output: o,
		URL:    url,
	}, nil
}

func (u *Uploader) s3(o *OutputFile, dest *UploadInfoS3) (*UploadResult, error) {
	startTime := time.Now()

	url, err := u.s3client.Upload(o.AbsPath(), dest.Bucket, dest.Key)
	if err != nil {
		return nil, err
	}

	return &UploadResult{
		Start:  startTime,
		Stop:   time.Now(),
		Method: UploadMethodS3,
		Output: o,
		URL:    url,
	}, nil
}
