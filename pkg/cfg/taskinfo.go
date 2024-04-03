package cfg

type TaskInfo struct {
	TaskName   string `toml:"task_name" comment:"name of a task of the same app"`
	EnvVarName string `toml:"env_var" comment:"name of an environment variable, when the task command is executed, is is set to to a file path.\n The temporary file contains the JSON encoded information about the task."`
}

func (t *TaskInfo) Validate() error {
	if err := validateTaskOrAppName(t.TaskName); err != nil {
		return fieldErrorWrap(err, "task_name")
	}

	if t.EnvVarName == "" {
		return newFieldError("can not be empty", "env_var")
	}

	return nil
}

func validateTaskInfos(infos []TaskInfo) error {
	for _, ti := range infos {
		if err := ti.Validate(); err != nil {
			return fieldErrorWrap(err, elementPathWithID("TaskInfo", ti.TaskName))
		}
	}

	return nil
}
