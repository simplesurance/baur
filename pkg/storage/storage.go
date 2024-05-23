// Package storage provides an interface for baur data storage implementations.
package storage

import (
	"context"
	"errors"
	"io"
	"time"
)

// ErrNotExist indicates that a record does not exist
var ErrNotExist = errors.New("does not exist")

// ErrExists indicates that the database or a record already exist.
var ErrExists = errors.New("already exists")

type InputFile struct {
	Path   string
	Digest string
}

type InputString struct {
	String string
	Digest string
}

type InputEnvVar struct {
	Name   string
	Digest string
}

type InputTaskInfo struct {
	Name   string
	Digest string
}

type Inputs struct {
	Files                []*InputFile
	Strings              []*InputString
	EnvironmentVariables []*InputEnvVar
	TaskInfo             []*InputTaskInfo
}

// UploadMethod is the method that was used to upload the object
type UploadMethod string

const (
	UploadMethodS3             UploadMethod = "s3"
	UploadMethodDockerRegistry UploadMethod = "docker"
	UploadMethodFileCopy       UploadMethod = "filecopy"
)

type Upload struct {
	URI                  string
	UploadStartTimestamp time.Time
	UploadStopTimestamp  time.Time
	Method               UploadMethod
}

type ArtifactType string

const (
	ArtifactTypeDocker ArtifactType = "docker"
	ArtifactTypeFile   ArtifactType = "file"
)

type Output struct {
	Name      string
	Type      ArtifactType
	Digest    string
	SizeBytes uint64
	Uploads   []*Upload
}

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
	Inputs  Inputs
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

	// SchemaVersion returns the version of the schema that the storage is
	// using.
	SchemaVersion(ctx context.Context) (int32, error)
	// RequiredSchemaVersion returns the schema version that the Storer
	// implementation requires.
	RequiredSchemaVersion() int32
	// IsCompatible verifies that the storage is compatible with the baur version
	IsCompatible(context.Context) error
	// Upgrade upgrades the schema to RequiredSchemaVersion().
	// If the database does not exist ErrNotExist is returned.
	Upgrade(ctx context.Context) error
	// Init initializes a storage, e.g. creating the database scheme.
	// If it already exist, ErrExist is returned.
	Init(context.Context) error

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

	// Inputs returns the inputs of a task run. If no records were found,
	// the method returns ErrNotExist.
	Inputs(ctx context.Context, taskRunID int) (*Inputs, error)
	Outputs(ctx context.Context, taskRunID int) ([]*Output, error)

	// CreateRelease creates a new release called releaseName, that consists of the the passed task runs.
	// Metadata is arbitrary data stored together with the release, it is
	// optional and can be nil.
	CreateRelease(_ context.Context, releaseName string, taskRunIDs []int, metadata io.Reader) error
	ReleaseExists(_ context.Context, name string) (bool, error)
}
