package baur

import (
	"fmt"
	"sort"

	"github.com/simplesurance/baur/v1/cfg"
)

// Task is a an execution step belonging to an app.
// A task has a set of Inputs that produce a set of outputs by executing it's
// Command.
type Task struct {
	RepositoryRoot string
	Directory      string

	AppName string

	Name             string
	Command          string
	UnresolvedInputs *cfg.Input
	Outputs          *cfg.Output
}

// NewTask returns a new Task.
func NewTask(cfg *cfg.Task, appName, repositoryRootdir, workingDir string) *Task {
	return &Task{
		RepositoryRoot:   repositoryRootdir,
		Directory:        workingDir,
		Outputs:          &cfg.Output,
		Command:          cfg.Command,
		Name:             cfg.Name,
		AppName:          appName,
		UnresolvedInputs: &cfg.Input,
	}
}

// ID returns <APP-NAME>.<TASK-NAME>
func (t *Task) ID() string {
	return fmt.Sprintf("%s.%s", t.AppName, t.Name)
}

// String returns ID()
func (t *Task) String() string {
	return t.ID()
}

// HasInputs returns true if Inputs are defined for the task
func (t *Task) HasInputs() bool {
	return !cfg.InputsAreEmpty(t.UnresolvedInputs)
}

// HHasOutputs returns true if outputs are defined for the task
func (t *Task) HasOutputs() bool {
	return len(t.Outputs.DockerImage) > 0 || len(t.Outputs.File) > 0
}

func SortTasksByID(tasks []*Task) {
	sort.Slice(tasks, func(i int, j int) bool {
		return tasks[i].ID() < tasks[j].ID()
	})
}
