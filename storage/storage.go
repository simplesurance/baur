package storage

import (
	"errors"
	"strings"
	"time"
)

// OutputType describes the type of an artifact
type OutputType string

const (
	//DockerOutput is a docker container artifact
	DockerOutput OutputType = "docker"
	//S3Output is a file artifact stored on S3
	S3Output OutputType = "s3"
)

// ErrNotExist indicates that a record does not exist
var ErrNotExist = errors.New("does not exist")

// Build represents a stored build
type Build struct {
	AppName          string
	StartTimeStamp   time.Time
	StopTimeStamp    time.Time
	TotalInputDigest string
	Outputs          []*Output
	Inputs           []*Input
}

// AppNameLower returns the app of the name in lowercase
func (b *Build) AppNameLower() string {
	return strings.ToLower(b.AppName)
}

// Output represents a build output
type Output struct {
	Name           string
	Type           OutputType
	URI            string
	Digest         string
	SizeBytes      int64
	UploadDuration time.Duration
}

// Input represents a source of an artifact
type Input struct {
	URL    string
	Digest string
}

// Storer is an interface for persisting informations about builds
type Storer interface {
	ListBuildsPerApp(appName string, maxResults int) ([]*Build, error)
	FindLatestAppBuildByDigest(appName, totalInputDigest string) (int64, error)
	Save(b *Build) error
	GetBuildWithoutInputs(id int64) (*Build, error)
}
