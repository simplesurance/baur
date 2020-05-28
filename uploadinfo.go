package baur

import "fmt"

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
	Bucket string
	Key    string
}

func (s *UploadInfoS3) Method() UploadMethod {
	return UploadMethodS3
}

func (s *UploadInfoS3) String() string {
	return fmt.Sprintf("s3://%s/%s", s.Bucket, s.Key)
}

type UploadInfoDocker struct {
	Registry   string
	Repository string
	Tag        string
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
	DestinationPath string
}

func (f *UploadInfoFileCopy) Method() UploadMethod {
	return UploadMethodFilecopy
}

func (f *UploadInfoFileCopy) String() string {
	return f.DestinationPath
}
