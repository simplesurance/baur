package baur

import "github.com/simplesurance/baur/v4/internal/digest"

// InputTask is a baur task whose information are used as input of another
// task.
type InputTask struct {
	task   *Task
	inputs *Inputs
}

func NewInputTask(task *Task, inputs *Inputs) *InputTask {
	return &InputTask{task: task, inputs: inputs}
}

func (i *InputTask) Digest() (*digest.Digest, error) {
	return i.inputs.Digest()
}

func (i *InputTask) TaskID() string {
	return i.task.ID
}

func (i *InputTask) String() string {
	return "task: " + i.task.ID
}
