package baur

// TaskInfo is information of task that is provided as Input to another one.
type TaskInfo struct {
	EnvVarName string
	Task       *Task
}
