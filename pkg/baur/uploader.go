package baur

import (
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"time"
)

type S3Uploader interface {
	Upload(filepath, bucket, key string) (string, error)
}

type DockerImgUploader interface {
	Upload(image, registryAddr, repository, tag string) (string, error)
}

type FileCopyUploader interface {
	Upload(src string, dst string) (string, error)
}

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

type UploadResult struct {
	Output Output
	URL    string
	Start  time.Time
	Stop   time.Time
	Method UploadMethod
}

type UploadStartFn func(Output, UploadInfo)

type UploadResultFn func(Output, *UploadResult)

func (u *Uploader) Upload(output Output, uploadStartCb UploadStartFn, resultCb UploadResultFn) error {
	switch o := output.(type) {
	case *OutputDockerImage:
		if o.UploadDestinations == nil {
			return errors.New("uploadDestination is nil")
		}

		for _, dest := range o.UploadDestinations {
			uploadStartCb(o, dest)

			result, err := u.DockerImage(o, dest)
			if err != nil {
				return fmt.Errorf("docker upload failed: %w", err)
			}

			resultCb(o, result)
		}

	case *OutputFile:
		for _, dest := range o.UploadsFilecopy {
			uploadStartCb(o, dest)

			result, err := u.FileCopy(o, dest)
			if err != nil {
				return fmt.Errorf("filecopy failed: %w", err)
			}

			resultCb(o, result)
		}

		for _, dest := range o.UploadsS3 {
			uploadStartCb(o, dest)

			result, err := u.S3(o, dest)
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

func (u *Uploader) DockerImage(o *OutputDockerImage, dest *UploadInfoDocker) (*UploadResult, error) {
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

func (u *Uploader) FileCopy(o *OutputFile, dest *UploadInfoFileCopy) (*UploadResult, error) {
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

func (u *Uploader) S3(o *OutputFile, dest *UploadInfoS3) (*UploadResult, error) {
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
