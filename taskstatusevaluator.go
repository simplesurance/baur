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

	additionalInputStr               string
	lookupAdditionalInputStrFallback string
}

// NewTaskStatusEvaluator returns a new TaskSNewTaskStatusEvaluator.
func NewTaskStatusEvaluator(
	repositoryDir string,
	store storage.Storer,
	inputResolver *InputResolver,
	additionalInputStr string,
	lookupAdditionalInputStrFallback string,
) *TaskStatusEvaluator {
	return &TaskStatusEvaluator{
		repositoryDir:                    repositoryDir,
		inputResolver:                    inputResolver,
		store:                            store,
		additionalInputStr:               additionalInputStr,
		lookupAdditionalInputStrFallback: lookupAdditionalInputStrFallback,
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

	inputs, err := t.inputResolver.Resolve(ctx, t.repositoryDir, task, t.additionalInputStr, t.lookupAdditionalInputStrFallback)
	if err != nil {
		return TaskStatusUndefined, nil, nil, fmt.Errorf("resolving inputs failed: %w", err)
	}

	totalInputDigest, err := inputs.TaskStatusDigest(ctx, task)
	if err != nil {
		return TaskStatusUndefined, nil, nil, fmt.Errorf("calculating total input digest failed: %w", err)
	}

	run, err := t.store.LatestTaskRunByDigest(ctx, task.AppName, task.Name, totalInputDigest.String())
	if err != nil {
		if err == storage.ErrNotExist {
			return TaskStatusExecutionPending, inputs, nil, nil
		}

		return TaskStatusUndefined, nil, nil, fmt.Errorf("querying storage for task run status failed: %w", err)
	}

	return TaskStatusRunExist, inputs, run, nil
}
