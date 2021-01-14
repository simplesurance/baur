package baur

import (
	"fmt"

	"github.com/simplesurance/baur/v1/cfg"
)

type UploadMethod int

const (
	UploadMethodS3 UploadMethod = iota
	UploadMethodFilecopy
	UploadMethodDocker
)

type UploadInfo interface {
	Method() UploadMethod
	String() string
}

type UploadInfoS3 struct {
	*cfg.S3Upload
}

func (s *UploadInfoS3) Method() UploadMethod {
	return UploadMethodS3
}

func (s *UploadInfoS3) String() string {
	return fmt.Sprintf("s3://%s/%s", s.Bucket, s.Key)
}

type UploadInfoDocker struct {
	*cfg.DockerImageRegistryUpload
}

func (d *UploadInfoDocker) Method() UploadMethod {
	return UploadMethodDocker
}

func (d *UploadInfoDocker) String() string {
	if d.Registry == "" {
		return fmt.Sprintf("%s/%s", d.Repository, d.Tag)
	}

	return fmt.Sprintf("%s/%s/%s", d.Registry, d.Repository, d.Tag)
}

type UploadInfoFileCopy struct {
	*cfg.FileCopy
}

func (f *UploadInfoFileCopy) Method() UploadMethod {
	return UploadMethodFilecopy
}

func (f *UploadInfoFileCopy) String() string {
	return f.Path
}
