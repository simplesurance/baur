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

	inputStr       string
	lookupInputStr string
}

// NewTaskStatusEvaluator returns a new TaskSNewTaskStatusEvaluator.
func NewTaskStatusEvaluator(
	repositoryDir string,
	store storage.Storer,
	inputResolver *InputResolver,
	inputStr string,
	lookupInputStr string,
) *TaskStatusEvaluator {
	return &TaskStatusEvaluator{
		repositoryDir:  repositoryDir,
		inputResolver:  inputResolver,
		store:          store,
		inputStr:       inputStr,
		lookupInputStr: lookupInputStr,
	}
}

// Status resolves the inputs of the task, calculates the total input
// digest and checks in the storage if a run record for the task and total input
// digest already exist.
// If TaskStatusExecutionPending is returned, the returned TaskRunWithID is nil.
func (t *TaskStatusEvaluator) Status(ctx context.Context, task *Task) (TaskStatus, *Inputs, *storage.TaskRunWithID, error) {
	var taskStatus TaskStatus
	var run *storage.TaskRunWithID

	inputFiles, err := t.inputResolver.Resolve(ctx, t.repositoryDir, task)
	if err != nil {
		return TaskStatusUndefined, nil, nil, fmt.Errorf("resolving inputs failed: %w", err)
	}

	inputs := NewInputs(InputAddStrIfNotEmpty(inputFiles, t.inputStr))

	taskStatus, run, err = t.getTaskStatus(ctx, inputs, task)
	if err != nil {
		return TaskStatusUndefined, nil, nil, err
	}

	if t.lookupInputStr == "" || taskStatus != TaskStatusExecutionPending {
		return taskStatus, inputs, run, err
	}

	inputsLookupStr := NewInputs(append(inputFiles, NewInputString(t.lookupInputStr)))
	taskStatus, run, err = t.getTaskStatus(ctx, inputsLookupStr, task)
	if err != nil {
		return TaskStatusUndefined, nil, nil, err
	}

	// inputs instead of inputsLookupInputStr must be returned, if the task
	// must be run it should be recorded with the inputStr not with the
	// lookupInputStr
	return taskStatus, inputs, run, err
}

func (t *TaskStatusEvaluator) getTaskStatus(ctx context.Context, inputs *Inputs, task *Task) (TaskStatus, *storage.TaskRunWithID, error) {
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
