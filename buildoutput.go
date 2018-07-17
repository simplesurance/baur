package baur

import (
	"github.com/simplesurance/baur/digest"
	"github.com/simplesurance/baur/upload"
)

// BuildOutput is an interface for build artifacts
type BuildOutput interface {
	Exists() bool
	UploadJob() (upload.Job, error)
	Name() string
	String() string
	LocalPath() string
	UploadDestination() string
	Digest() (*digest.Digest, error)
	Size(*BuildOutputBackends) (int64, error)
}
