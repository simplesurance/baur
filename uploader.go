package baur

import (
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
		if o.UploadDestination == nil {
			break
		}

		for _, dest := range o.UploadDestination {
			uploadStartCb(o, dest)

			result, err := u.DockerImage(o, dest)
			if err != nil {
				return fmt.Errorf("docker upload failed: %w", err)
			}

			resultCb(o, result)
		}

	case *OutputFile:
		if o.UploadsFilecopy != nil {
			uploadStartCb(o, o.UploadsFilecopy)

			result, err := u.FileCopy(o)
			if err != nil {
				return fmt.Errorf("filecopy failed: %w", err)
			}

			resultCb(o, result)
		}

		if o.UploadsS3 != nil {
			uploadStartCb(o, o.UploadsS3)

			result, err := u.S3(o)
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

func (u *Uploader) FileCopy(o *OutputFile) (*UploadResult, error) {
	startTime := time.Now()

	destFile := filepath.Join(o.UploadsFilecopy.DestinationPath, filepath.Base(o.AbsPath))

	url, err := u.filecopyUploader.Upload(o.AbsPath, destFile)
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

func (u *Uploader) S3(o *OutputFile) (*UploadResult, error) {
	startTime := time.Now()

	url, err := u.s3client.Upload(o.AbsPath, o.UploadsS3.Bucket, o.UploadsS3.Key)
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
