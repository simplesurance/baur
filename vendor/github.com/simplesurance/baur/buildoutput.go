package baur

import (
	"github.com/simplesurance/baur/digest"
	"github.com/simplesurance/baur/upload/scheduler"
)

// BuildOutput is an interface for build artifacts
type BuildOutput interface {
	Exists() bool
	UploadJob() (scheduler.Job, error)
	Name() string
	String() string
	LocalPath() string
	UploadDestination() string
	Digest() (*digest.Digest, error)
	Size(*BuildOutputBackends) (int64, error)
	Type() string
}
