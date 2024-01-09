package baur

import (
	"context"
	"errors"
	"fmt"

	"github.com/simplesurance/baur/v3/pkg/storage"
)

// TaskStatusEvaluator determines if a task already run with the same set of
// inputs in the past.
type TaskStatusEvaluator struct {
	repositoryDir string

	inputResolver *InputResolver
	store         storage.Storer

	inputStr       []string
	lookupInputStr string
}

// NewTaskStatusEvaluator returns a new TaskSNewTaskStatusEvaluator.
func NewTaskStatusEvaluator(
	repositoryDir string,
	store storage.Storer,
	inputResolver *InputResolver,
	inputStr []string,
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

// Status resolves the inputs of the task, calculates the total input digest
// and checks in the storage if a run record for the task and total input
// digest already exist.
// If TaskStatusExecutionPending is returned, the returned TaskRunWithID is nil.
func (t *TaskStatusEvaluator) Status(ctx context.Context, task *Task) (TaskStatus, *Inputs, *storage.TaskRunWithID, error) {
	var taskStatus TaskStatus
	var run *storage.TaskRunWithID

	inputFiles, err := t.inputResolver.Resolve(ctx, t.repositoryDir, task)
	if err != nil {
		return TaskStatusUndefined, nil, nil, err
	}

	inputs := NewInputs(append(AsInputStrings(t.inputStr...), inputFiles...))
	taskStatus, run, err = t.getTaskStatus(ctx, inputs, task)
	if err != nil {
		return TaskStatusUndefined, nil, nil, err
	}

	if t.lookupInputStr == "" || taskStatus != TaskStatusExecutionPending {
		return taskStatus, inputs, run, err
	}

	inputsLookupStr := NewInputs(append(AsInputStrings(t.lookupInputStr), inputFiles...))
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
		if errors.Is(err, storage.ErrNotExist) {
			return TaskStatusExecutionPending, nil, nil
		}

		return TaskStatusUndefined, nil, fmt.Errorf("querying storage for task run status failed: %w", err)
	}

	return TaskStatusRunExist, run, nil
}
