package build

import (
	"time"
)

// Result result of a build job
type Result struct {
	Job   *Job
	Error error

	StartTs  time.Time
	StopTs   time.Time
	ExitCode int
	Output   string
}

// Job describes a build job
type Job struct {
	Application string
	Directory   string
	Command     string
	UserData    interface{}
}

// Builder is an interface for builders
type Builder interface {
	Start()
}
