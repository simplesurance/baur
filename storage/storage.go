package storage

import (
	"context"
	"errors"
	"time"
)

// ErrNotExist indicates that a record does not exist
var ErrNotExist = errors.New("does not exist")

type Input struct {
	URI    string
	Digest string
}

// UploadMethod is the method that was used to upload the object
type UploadMethod string

const (
	UploadMethodS3             UploadMethod = "s3"
	UploadMethodDockerRegistry UploadMethod = "docker"
	UploadMethodFileCopy       UploadMethod = "filecopy"
)

// Upload contains informations about an output upload
type Upload struct {
	URI                  string
	UploadStartTimestamp time.Time
	UploadStopTimestamp  time.Time
	Method               UploadMethod
}

// ArtifactType describes the type of an artifact
type ArtifactType string

const (
	ArtifactTypeDocker ArtifactType = "docker"
	ArtifactTypeFile   ArtifactType = "file"
)

// Output represents a task output
type Output struct {
	Name      string
	Type      ArtifactType
	Digest    string
	SizeBytes uint64
	Uploads   []*Upload
}

// Result is the result of a task run
type Result string

const (
	ResultSuccess Result = "success"
	ResultFailure Result = "failure"
)

type TaskRun struct {
	ApplicationName  string
	TaskName         string
	VCSRevision      string
	VCSIsDirty       bool
	StartTimestamp   time.Time
	StopTimestamp    time.Time
	TotalInputDigest string
	Result           Result
}

type TaskRunFull struct {
	TaskRun
	Inputs  []*Input
	Outputs []*Output
}

type TaskRunWithID struct {
	ID int
	TaskRun
}

const (
	NoLimit uint = 0
)

// Storer is an interface for storing and retrieving baur task runs
type Storer interface {
	Close() error

	// Init initializes a storage, e.g. creating the database scheme
	Init(context.Context) error
	// IsCompatible verifies that the storage is compatible with the baur version
	IsCompatible(context.Context) error

	SaveTaskRun(context.Context, *TaskRunFull) (id int, err error)
	LatestTaskRunByDigest(ctx context.Context, appName, taskName, totalInputDigest string) (*TaskRunWithID, error)

	TaskRun(ctx context.Context, id int) (*TaskRunWithID, error)
	// TaskRuns queries the storage for runs that match the filters.
	// A limit value of 0 will return all results.
	// The found results are passed in iterative manner to the callback
	// function. When the callback function returns an error, the iteration
	// stops.
	// When no matching records exist, the method returns ErrNotExist.
	TaskRuns(ctx context.Context,
		filters []*Filter,
		sorters []*Sorter,
		limit uint,
		callback func(*TaskRunWithID) error,
	) error

	Inputs(ctx context.Context, taskRunID int) ([]*Input, error)
	Outputs(ctx context.Context, taskRunID int) ([]*Output, error)
}
