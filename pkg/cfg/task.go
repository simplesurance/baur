package cfg

import (
	"github.com/simplesurance/baur/v1/pkg/cfg/resolver"
)

// Task is a task section
type Task struct {
	Name     string   `toml:"name"`
	Command  []string `toml:"command" comment:"Command to execute.\n The first element is the command, the following it's arguments.\n If the command element contains no path seperators,\n the path is looked up via the $PATH environment variable."`
	Includes []string `toml:"includes" comment:"Input or Output includes that the task inherits.\n Includes are specified in the format <filepath>#<ID>.\n Paths are relative to the application directory."`
	Input    Input    `toml:"Input" comment:"Specification of task inputs like source files, Makefiles, etc"`
	Output   Output   `toml:"Output" comment:"Specification of task outputs produced by the Task.command"`

	// multiple include sections of the same file can be included, use a map
	// instead of a slice to act as a Set datastructure
	cfgFiles map[string]struct{}
}

func (t *Task) addCfgFilepath(path string) {
	if path == "" {
		panic("path is empty")
	}

	t.cfgFiles[path] = struct{}{}
}

// Filepaths returns a list of all parsed config files.
// This is the app config file and files of the included sections.
func (t *Task) Filepaths() []string {
	result := make([]string, 0, len(t.cfgFiles))

	for p := range t.cfgFiles {
		result = append(result, p)
	}

	return result
}

func (t *Task) GetCommand() []string {
	return t.Command
}
func (t *Task) GetName() string {
	return t.Name
}

func (t *Task) GetIncludes() *[]string {
	return &t.Includes
}

func (t *Task) GetInput() *Input {
	return &t.Input
}

func (t *Task) GetOutput() *Output {
	return &t.Output
}

func (t *Task) resolve(resolvers resolver.Resolver) error {
	var err error

	for i, elem := range t.Command {
		if t.Command[i], err = resolvers.Resolve(elem); err != nil {
			return fieldErrorWrap(err, "Command")
		}
	}

	if err := t.Input.resolve(resolvers); err != nil {
		return fieldErrorWrap(err, "Input")
	}

	if err := t.Output.Resolve(resolvers); err != nil {
		return fieldErrorWrap(err, "Output")
	}

	return nil
}
