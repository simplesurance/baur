// Package seq provides a sequential builder. All jobs are build sequentialy,
// nothing fancy.
package seq

import (
	"time"

	"github.com/simplesurance/baur/build"
	"github.com/simplesurance/baur/exec"
)

// Builder represents a sequential builder
type Builder struct {
	jobs       []*build.Job
	statusChan chan<- *build.Result
}

// New returns a new builder instance
func New(jobs []*build.Job, status chan<- *build.Result) build.Builder {
	return &Builder{
		jobs:       jobs,
		statusChan: status,
	}
}

// Start starts building applications
func (b *Builder) Start() {
	// TODO: support canceling the build via ctx.Done()
	for _, j := range b.jobs {
		startTime := time.Now()

		out, exitCode, err := exec.Command(j.Directory, j.Command)
		res := build.Result{
			Job:      j,
			Error:    err,
			Duration: time.Since(startTime),
			ExitCode: exitCode,
			Output:   out,
		}

		b.statusChan <- &res
	}

	close(b.statusChan)
}
