package baur

import (
	"context"
	"fmt"

	"github.com/simplesurance/baur/v1/storage"
)

// TaskTaskStatusEvaluator evaluates the status of a task.
type TaskStatusEvaluator struct {
	repositoryDir string

	inputResolver *InputResolver
	store         storage.Storer

	inputStr    string
	altInputStr string
}

// NewTaskStatusEvaluator returns a new TaskSNewTaskStatusEvaluator.
func NewTaskStatusEvaluator(
	repositoryDir string,
	store storage.Storer,
	inputResolver *InputResolver,
	inputStr string,
	altInputStr string,
) *TaskStatusEvaluator {
	return &TaskStatusEvaluator{
		repositoryDir: repositoryDir,
		inputResolver: inputResolver,
		store:         store,
		inputStr:      inputStr,
		altInputStr:   altInputStr,
	}
}

// Status resolves the inputs of the task, calculates the total input
// digest and checks in the storage if a run record for the task and total input
// digest already exist.
// If TaskStatusInputsUndefined is returned, the returned Inputs slice and TaskRunWithID are nil.
// If TaskStatusExecutionPending is returned, the returned TaskRunWithID is nil.
func (t *TaskStatusEvaluator) Status(ctx context.Context, task *Task) (TaskStatus, *Inputs, *storage.TaskRunWithID, error) {
	if !task.HasInputs() {
		return TaskStatusInputsUndefined, nil, nil, nil
	}

	inputs, err := t.inputResolver.Resolve(ctx, t.repositoryDir, task)
	if err != nil {
		return TaskStatusUndefined, nil, nil, fmt.Errorf("resolving inputs failed: %w", err)
	}

	var taskStatus TaskStatus
	var run *storage.TaskRunWithID

	inputs.SetInputString(t.inputStr)
	taskStatus, run, err = t.getTaskStatus(ctx, inputs, task)

	if err != nil {
		return TaskStatusUndefined, nil, nil, err
	}

	if taskStatus == TaskStatusExecutionPending && t.altInputStr != "" {
		inputs.SetInputString(t.altInputStr)
		taskStatus, run, err = t.getTaskStatus(ctx, inputs, task)

		inputs.SetInputString(t.inputStr)
	}

	return taskStatus, inputs, run, err
}

func (t *TaskStatusEvaluator) getTaskStatus(ctx context.Context, inputs Input, task *Task) (TaskStatus, *storage.TaskRunWithID, error) {
	totalInputDigest, err := inputs.Digest()
	if err != nil {
		return TaskStatusUndefined, nil, fmt.Errorf("calculating total input digest failed: %w", err)
	}

	run, err := t.store.LatestTaskRunByDigest(ctx, task.AppName, task.Name, totalInputDigest.String())
	if err != nil {
		if err == storage.ErrNotExist {
			return TaskStatusExecutionPending, nil, nil
		}

		return TaskStatusUndefined, nil, fmt.Errorf("querying storage for task run status failed: %w", err)
	}

	return TaskStatusRunExist, run, nil
}
