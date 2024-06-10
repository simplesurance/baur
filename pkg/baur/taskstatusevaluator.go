package baur

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/simplesurance/baur/v4/pkg/storage"
)

// TaskStatusEvaluator determines if a task already run with the same set of
// inputs in the past.
type TaskStatusEvaluator struct {
	repositoryDir string

	inputResolver *InputResolver
	store         storage.Storer

	lookupInputStr string
}

func NewTaskStatusEvaluator(
	repositoryDir string,
	store storage.Storer,
	inputResolver *InputResolver,
	lookupInputStr string,
) *TaskStatusEvaluator {
	return &TaskStatusEvaluator{
		repositoryDir:  repositoryDir,
		inputResolver:  inputResolver,
		store:          store,
		lookupInputStr: lookupInputStr,
	}
}

func replaceInputStrings(inputs *Inputs, replacement []Input) *Inputs {
	result := slices.Clone(inputs.Inputs())
	result = slices.DeleteFunc(result, func(input Input) bool {
		_, ok := input.(*InputString)
		return ok
	})
	return NewInputs(append(result, replacement...))
}

// Status resolves the inputs of the task, calculates the total input digest
// and checks in the storage if a run record for the task and total input
// digest already exist.
// If a run exist it returns it and TaskStatusExecutionPending.
// If no run exist it returns a nil storage.TaskRunWithID and TaskStatusExecutionPending.
//
// The method caches results of successful (err==nil) calls per Task. Running
// it multiple times for the same task returns the same result.
func (t *TaskStatusEvaluator) Status(ctx context.Context, task *Task) (TaskStatus, *Inputs, *storage.TaskRunWithID, error) {
	if len(t.lookupInputStr) != 0 && len(task.UnresolvedInputs.TaskInfos) > 0 {
		return TaskStatusUndefined, nil, nil,
			fmt.Errorf("task %q defines TaskInfo Inputs, using them when specifying '--lookup-input-str' is unsupported", task.ID)
	}

	inputs, err := t.inputResolver.Resolve(ctx, task)
	if err != nil {
		return TaskStatusUndefined, nil, nil, err
	}

	run, err := t.getTaskStatus(ctx, task, inputs)
	if err == nil {
		return TaskStatusRunExist, inputs, run, nil
	}

	if !errors.Is(err, storage.ErrNotExist) {
		return TaskStatusUndefined, nil, nil, err
	}

	if t.lookupInputStr == "" {
		return TaskStatusExecutionPending, inputs, run, nil
	}

	inputsWithLookupStr := replaceInputStrings(inputs, AsInputStrings(t.lookupInputStr))
	run, err = t.getTaskStatus(ctx, task, inputsWithLookupStr)
	if err != nil {
		if errors.Is(err, storage.ErrNotExist) {
			// inputs instead of inputsLookupInputStr must be returned, if the task
			// must be run it should be recorded with the inputStr not with the
			// lookupInputStr
			return TaskStatusExecutionPending, inputs, run, nil
		}

		return TaskStatusUndefined, nil, nil, err
	}

	return TaskStatusRunExist, inputsWithLookupStr, run, nil
}

func (t *TaskStatusEvaluator) getTaskStatus(ctx context.Context, task *Task, inputs *Inputs) (*storage.TaskRunWithID, error) {
	totalInputDigest, err := inputs.Digest()
	if err != nil {
		return nil, fmt.Errorf("calculating total input digest failed: %w", err)
	}

	run, err := t.store.LatestTaskRunByDigest(ctx, task.AppName, task.Name, totalInputDigest.String())
	if err != nil {
		return nil, fmt.Errorf("querying storage for task run status failed: %w", err)
	}

	return run, nil
}
