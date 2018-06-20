package storage

import (
	"strings"
	"time"

	"github.com/simplesurance/baur/digest"
)

// ArtifactType describes the type of an artifact
type ArtifactType string

const (
	//DockerArtifact is a docker container artifact
	DockerArtifact ArtifactType = "docker"
	//S3Artifact is a file artifact stored on S3
	S3Artifact ArtifactType = "s3"
)

// Build represents a stored build
type Build struct {
	AppName        string
	StartTimeStamp time.Time
	StopTimeStamp  time.Time
	Artifacts      []*Artifact
	TotalSrcDigest string
	Sources        []*Source
}

// AppNameLower returns the app of the name in lowercase
func (b *Build) AppNameLower() string {
	return strings.ToLower(b.AppName)
}

// Artifact represents a stored artifact
type Artifact struct {
	Name           string
	Type           ArtifactType
	URI            string
	Digest         digest.Digest
	SizeBytes      int64
	UploadDuration time.Duration
}

// Source represents a source of an artifact
type Source struct {
	RelativePath string
	Digest       string
}

// Storer is an interface for persisting informations about builds
type Storer interface {
	ListBuildsPerApp(appName string, maxResults int) ([]*Build, error)
	Save(b *Build) error
}
