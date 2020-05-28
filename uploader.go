package baur

import (
	"fmt"
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

		uploadStartCb(o, o.UploadDestination)

		result, err := u.DockerImage(o)
		if err != nil {
			return err
		}

		resultCb(o, result)

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

func (u *Uploader) DockerImage(o *OutputDockerImage) (*UploadResult, error) {
	startTime := time.Now()

	url, err := u.dockerclient.Upload(
		o.ImageID,
		o.UploadDestination.Registry,
		o.UploadDestination.Repository,
		o.UploadDestination.Tag,
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

	url, err := u.filecopyUploader.Upload(o.AbsPath, o.UploadsFilecopy.DestinationPath)
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
