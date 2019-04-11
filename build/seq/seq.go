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

		// TODO: add the application name as .DebugPrefix()
		cmdRes, err := exec.ShellCommand(j.Command).Directory(j.Directory).Run()
		res := build.Result{
			Job:      j,
			Error:    err,
			StartTs:  startTime,
			StopTs:   time.Now(),
			ExitCode: cmdRes.ExitCode,
			Output:   cmdRes.StrOutput(),
		}

		b.statusChan <- &res
	}

	close(b.statusChan)
}
