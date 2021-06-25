package cfg

import (
	"github.com/simplesurance/baur/v2/internal/deepcopy"
)

// TaskInclude is a reusable Tasks definition
type TaskInclude struct {
	IncludeID string `toml:"include_id" comment:"identifier of the include"`

	Name     string   `toml:"name" comment:"Task name"`
	Command  []string `toml:"command" comment:"Command to execute. The first element is the command, the following its arguments.\n If the command element contains no path seperators, its path is looked up via the $PATH environment variable."`
	Includes []string `toml:"includes" comment:"Input or Output includes that the task inherits.\n Includes are specified in the format <filepath>#<ID>.\n Paths are relative to the include file location."`
	Input    Input    `toml:"Input" comment:"Specification of task inputs like source files, Makefiles, etc"`
	Output   Output   `toml:"Output" comment:"Specification of task outputs produced by the Task.command"`

	cfgFiles map[string]struct{}
}

func (t *TaskInclude) addCfgFilepath(path string) {
	t.cfgFiles[path] = struct{}{}
}

func (t *TaskInclude) command() []string {
	return t.Command
}

func (t *TaskInclude) name() string {
	return t.Name
}

func (t *TaskInclude) includes() *[]string {
	return &t.Includes
}

func (t *TaskInclude) input() *Input {
	return &t.Input
}

func (t *TaskInclude) output() *Output {
	return &t.Output
}

func (t *TaskInclude) validate() error {
	if err := validateIncludeID(t.IncludeID); err != nil {
		if t.IncludeID != "" {
			err = fieldErrorWrap(err, t.IncludeID)
		}
		return err
	}

	return taskValidate(t)
}

// toTask converts the TaskInclude to a Task.
// All fields are copied.
func (t *TaskInclude) toTask() *Task {
	var result Task

	result.Name = t.Name
	result.Command = make([]string, len(t.Command))
	copy(result.Command, t.Command)

	result.cfgFiles = make(map[string]struct{}, len(result.cfgFiles))
	for k, v := range t.cfgFiles {
		result.cfgFiles[k] = v
	}

	deepcopy.MustCopy(t.Input, &result.Input)
	deepcopy.MustCopy(t.Output, &result.Output)

	return &result
}
