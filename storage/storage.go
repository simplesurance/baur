package storage

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// ArtifactType describes the type of an artifact
type ArtifactType string

const (
	//DockerArtifact is a docker image artifact
	DockerArtifact ArtifactType = "docker"
	//FileArtifact is a file artifact
	FileArtifact ArtifactType = "file"
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
	Build
	Duration time.Duration
}

// NameLower returns the app of the name in lowercase
func (a *Application) NameLower() string {
	return strings.ToLower(a.Name)
}

// Upload contains informations about an output upload
type Upload struct {
	ID             int
	UploadDuration time.Duration
	URI            string
}

// Output represents a build output
type Output struct {
	Name      string
	Type      ArtifactType
	Digest    string
	SizeBytes int64
	Upload    Upload
}

// Field represents data fields that can be used in sort and filter operations
type Field int

// Defines the available data fields
const (
	FieldUndefined Field = iota
	FieldApplicationName
	FieldBuildDuration
	FieldBuildStartTime
	FieldBuildID
)

func (f Field) String() string {
	switch f {
	case FieldApplicationName:
		return "FieldApplicationName"
	case FieldBuildDuration:
		return "FieldBuildDuration"
	case FieldBuildStartTime:
		return "FieldBuildStartTime"
	case FieldBuildID:
		return "FieldBuildID"
	default:
		return "FieldUndefined"
	}
}

// Input represents a source of an artifact
type Input struct {
	URI    string
	Digest string
}

// Filter specifies filter operatons for queries
type Filter struct {
	Field    Field
	Operator Op
	Value    interface{}
}

// Op describes the filter operator
type Op int

const (
	// OpEQ represents an equal (=) operator
	OpEQ Op = iota
	// OpGT represents a greater than (>) operator
	OpGT
	// OpLT represents a smaller than (<) operator
	OpLT
	// OpIN represents a In operator, works like the SQL IN operator, the
	// corresponding Value field in The filter struct must be a slice
	OpIN
)

func (o Op) String() string {
	switch o {
	case OpEQ:
		return "OpEQ"
	case OpGT:
		return "OpGT"
	default:
		return "OpUndefined"
	}
}

// Order specifies the sort order
type Order int

const (
	// SortInvalid represents an invalid sort value
	SortInvalid Order = iota
	// OrderAsc sorts ascending
	OrderAsc
	// OrderDesc sorts descending
	OrderDesc
)

func (s Order) String() string {
	switch s {
	case OrderAsc:
		return "asc"
	case OrderDesc:
		return "desc"
	default:
		return "invalid"
	}
}

//OrderFromStr converts a string to an Order
func OrderFromStr(s string) (Order, error) {
	switch strings.ToLower(s) {
	case "asc":
		return OrderAsc, nil
	case "desc":
		return OrderDesc, nil
	default:
		return SortInvalid, errors.New("undefined order")
	}
}

// Sorter specifies how the result of queries should be sorted
type Sorter struct {
	Field Field
	Order Order
}

// String return the string representation
func (s *Sorter) String() string {
	return fmt.Sprintf("%s-%s", s.Field, s.Order)
}

// Storer is an interface for persisting informations about builds
type Storer interface {
	Init() error
	Save(b *Build) error

	GetApps() ([]*Application, error)

	GetSameTotalInputDigestsForAppBuilds(appName string, startTs time.Time) (map[string][]int, error)
	GetLatestBuildByDigest(appName, totalInputDigest string) (*BuildWithDuration, error)
	GetBuildOutputs(buildID int) ([]*Output, error)

	// GetBuildWithoutInputs returns a single build, if no build with the ID
	// exist ErrNotExist is returned
	GetBuildWithoutInputs(id int) (*BuildWithDuration, error)
	GetBuildsWithoutInputs(filters []*Filter, sorters []*Sorter) ([]*BuildWithDuration, error)
}
