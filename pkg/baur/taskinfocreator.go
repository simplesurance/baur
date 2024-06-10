package baur

import (
	"context"
	"fmt"

	"github.com/simplesurance/baur/v4/pkg/cfg"
	"github.com/simplesurance/baur/v4/pkg/storage"
)

/* The implementation has the followig caveats:
   - Using TaskInfoCreator with input lookup strings is unsupported
   - TaskInfoCreator.CreateFileContent queries a Task Run again that might have
     been queried already before when the status of the tasks was evaluated.
     This is unnecessary.
   - TaskInfoCreator.CreateFileContent retrieves in 2 db operations first a
     whole run and then it's outputs, it only uses the runId and Outputs
     though.
   - It depends on the execution order of tasks if the outputs referenced in
     the TaskInfo are existing ones, queried from the database or are
     non-existing ones that still need to be build. (FIXME: add this comment to
   - The TaskInfoCreator stores a TaskStatusEvaluator reference, which
     references an InputResolver instance. This means that the inputResolver
     cache is hold much longer in the memory then before, which increases the
     memory usage when running tasks.
   - Inputs for the same Task are resolved multiple times, when the same task
     is used as TaskInfo for multiple builds. This might be negligible because
     resolving actual inputs files uses a cache and also the digests are only
     calculated one time because of the InputFileSingletonCache.

*/

type TaskInfoCreator struct {
	store           storage.Storer
	statusEvaluator *TaskStatusEvaluator
}

func NewTaskInfoCreator(storage storage.Storer, statusEvaluator *TaskStatusEvaluator) *TaskInfoCreator {
	return &TaskInfoCreator{
		store:           storage,
		statusEvaluator: statusEvaluator,
	}
}

func (c *TaskInfoCreator) CreateFileContent(ctx context.Context, task *Task) (*TaskInfoFile, error) {
	var result TaskInfoFile

	// TODO: Instead of 2 database queries, to first retrieve the TaskRun
	// via the statusEvaluator and then the outputs with another query, do
	// a db single query!

	// TODO: The Status of a task and it's outputs might currently be
	//       retrieved multiple times. This happens if a TaskInfo task is
	//       also part of the tasks that can be run AND when the same task
	//       is used for multiple TaskInfos. Optimize it do retrieve the
	//       status and outputs only 1x per task.

	status, inputs, taskRun, err := c.statusEvaluator.Status(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("retrieving task status of %q failed: %w", task.ID, err)
	}

	switch status {
	case TaskStatusRunExist:
		outputs, err := c.store.Outputs(ctx, taskRun.ID)
		if err != nil {
			return nil, fmt.Errorf("retrieving outputs from storage for %q task-run: %d failed: %w", task.ID, taskRun.ID, err)
		}
		result.Outputs = storageOutputsToTaskInfoOutput(outputs)
	case TaskStatusExecutionPending:
		result.Outputs = cfgOutputsToTaskInfoOutput(task.Outputs)
	default:
		return nil, fmt.Errorf("BUG: statusEvaluator returned no error and status: %s", status)
	}

	digest, err := inputs.Digest()
	if err != nil {
		return nil, fmt.Errorf("BUG: retrieving total input digest for %s failed, this should not happen digest must have been calculated already and cached: %w", task.ID, err)
	}

	result.TotalInputDigest = digest.String()
	result.AppDir = task.Directory

	return &result, nil
}

func storageOutputsToTaskInfoOutput(outputs []*storage.Output) []*taskInfoOutput {
	result := make([]*taskInfoOutput, 0, len(outputs)) // allocate space can be insufficient, one output can have >1 uploads

	for _, out := range outputs {
		for _, upload := range out.Uploads {
			result = append(result, &taskInfoOutput{
				URI: upload.URI,
			})
		}
	}
	return result
}

func cfgOutputsToTaskInfoOutput(outputs *cfg.Output) []*taskInfoOutput {
	// allocated space can be insufficient, one output can have >1 uploads
	result := make([]*taskInfoOutput, 0, len(outputs.File)+len(outputs.DockerImage))

	for _, fileOut := range outputs.File {
		for _, fileCopy := range fileOut.FileCopy {
			uri := (&UploadInfoFileCopy{&fileCopy}).String()
			result = append(result, &taskInfoOutput{URI: uri})
		}

		for _, s3Upload := range fileOut.S3Upload {
			uri := (&UploadInfoS3{&s3Upload}).String()
			result = append(result, &taskInfoOutput{URI: uri})
		}
	}

	for _, dockerImage := range outputs.DockerImage {
		for _, out := range dockerImage.RegistryUpload {
			uri := (&UploadInfoDocker{&out}).String()
			result = append(result, &taskInfoOutput{URI: uri})
		}
	}

	return result
}
