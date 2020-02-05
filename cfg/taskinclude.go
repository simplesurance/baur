package cfg

// TaskInclude is a reusable Tasks definition
type TaskInclude struct {
	IncludeID string `toml:"include_id" comment:"identifier of the include"`

	Name     string   `toml:"name" comment:"Identifies the task, currently the name must be 'build'."`
	Command  string   `toml:"command" comment:"Command that the task executes"`
	Includes []string `toml:"includes" comment:"Input or Output includes that the task inherits.\n Includes are specified in the format <filepath>#<ID>.\n Paths are relative to the include file location.\n Valid variables: $ROOT"`
	Input    Input    `toml:"Input" comment:"Specification of task inputs like source files, Makefiles, etc"`
	Output   Output   `toml:"Output" comment:"Specification of task outputs produced by the Task.command"`
}

func (t *TaskInclude) GetCommand() string {
	return t.Command
}

func (t *TaskInclude) GetName() string {
	return t.Name
}

func (t *TaskInclude) GetIncludes() *[]string {
	return &t.Includes
}

func (t *TaskInclude) GetInput() *Input {
	return &t.Input
}

func (t *TaskInclude) GetOutput() *Output {
	return &t.Output
}

func (t *TaskInclude) Validate() error {
	if err := validateIncludeID(t.IncludeID); err != nil {
		if t.IncludeID != "" {
			err = FieldErrorWrap(err, t.IncludeID)
		}
		return err
	}

	if err := TaskValidate(t); err != nil {
		return err
	}

	return nil
}
