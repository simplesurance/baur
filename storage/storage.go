package storage

import (
	"database/sql"
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

// VCSState contains informations about the VCS at the time of the build
type VCSState struct {
	CommitID string
	IsDirty  bool
}

// Application stores the name of the Application
type Application struct {
	ID   int
	Name string
}

// Build represents a stored build
type Build struct {
	ID               int
	Application      Application
	VCSState         VCSState
	StartTimeStamp   time.Time
	StopTimeStamp    time.Time
	TotalInputDigest string
	Outputs          []*Output
	Inputs           []*Input
}

// BuildWithDuration adds duration to a Build
type BuildWithDuration struct {
	Build    Build
	Duration float64
}

// NameLower returns the app of the name in lowercase
func (a *Application) NameLower() string {
	return strings.ToLower(a.Name)
}

// Upload contains informations about an output upload
type Upload struct {
	ID             int
	UploadDuration time.Duration
	URL            string
}

// Output represents a build output
type Output struct {
	Name      string
	Type      OutputType
	Digest    string
	SizeBytes int64
	Upload    Upload
}

// Input represents a source of an artifact
type Input struct {
	URL    string
	Digest string
}

// RowScanFunc should run rows.Scan and return a value for that row
type RowScanFunc func(rows *sql.Rows) (interface{}, error)

// Storer is an interface for persisting informations about builds
type Storer interface {
	GetLatestBuildByDigest(appName, totalInputDigest string) (*Build, error)
	Save(b *Build) error
	GetApps() ([]*Application, error)
	GetSameTotalInputDigestsForAppBuilds(appName string, startTs time.Time) (map[string][]int, error)
	GetBuildWithoutInputs(id int) (*Build, error)
	GetBuildsWithoutInputs(buildIDs []int) ([]*Build, error)
	GetBuildOutputs(buildID int) ([]*Output, error)
	GetBuilds(filters []CanFilter, sorters []CanSort) ([]*BuildWithDuration, error)
}
