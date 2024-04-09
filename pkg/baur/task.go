package baur

import (
	"fmt"
	"sort"

	"github.com/simplesurance/baur/v3/pkg/cfg"
)

// Task is a an execution step belonging to an app.
// A task has a set of Inputs that produce a set of outputs by executing it's
// Command.
type Task struct {
	// ID is the unique identifier of the task, in the format.
	// <APP-NAME>.<TASK-NAME>.
	ID string

	RepositoryRoot string
	Directory      string

	AppName string

	Name             string
	Command          []string
	UnresolvedInputs *cfg.Input
	Outputs          *cfg.Output
	CfgFilepaths     []string
}

// NewTask returns a new Task.
func NewTask(cfg *cfg.Task, appName, repositoryRootdir, workingDir string) *Task {
	return &Task{
		ID:               fmt.Sprintf("%s.%s", appName, cfg.Name),
		RepositoryRoot:   repositoryRootdir,
		Directory:        workingDir,
		Outputs:          &cfg.Output,
		CfgFilepaths:     cfg.Filepaths(),
		Command:          cfg.Command,
		Name:             cfg.Name,
		AppName:          appName,
		UnresolvedInputs: &cfg.Input,
	}
}

// String returns ID()
func (t *Task) String() string {
	return t.ID
}

// HasInputs returns true if Inputs are defined for the task
func (t *Task) HasInputs() bool {
	return !t.UnresolvedInputs.IsEmpty()
}

// HasOutputs returns true if outputs are defined for the task
func (t *Task) HasOutputs() bool {
	return len(t.Outputs.DockerImage) > 0 || len(t.Outputs.File) > 0
}

// SortTasksByID sorts the tasks slice by task IDs.
func SortTasksByID(tasks []*Task) {
	sort.Slice(tasks, func(i int, j int) bool {
		return tasks[i].ID < tasks[j].ID
	})
}
