package upload

import (
	"context"
	"time"
)

// S3Uploader is an interface for S3 uploader
type S3Uploader interface {
	Upload(file, dest string) (string, error)
}

// DockerUploader is an interface for docker uploader
type DockerUploader interface {
	Upload(ctx context.Context, image, dest string) (string, error)
}

// Manager is an interface for upload managers
type Manager interface {
	Add(Job)
	Start()
	Stop()
}

// Result result of an upload attempt
type Result struct {
	Err      error
	URL      string
	Duration time.Duration
	Job      Job
}
