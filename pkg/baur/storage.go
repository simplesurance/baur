package baur

import (
	"context"
	"fmt"

	"github.com/simplesurance/baur/v5/internal/vcs/git"
	"github.com/simplesurance/baur/v5/pkg/storage"
)

// StoreRun stores the result of a task run in a baur storage.
func StoreRun(
	ctx context.Context,
	storer storage.Storer,
	gitRepo *git.Repository,
	task *Task,
	inputs *Inputs,
	runResult *RunResult,
	uploads []*UploadResult,
) (int, error) {
	var commitID string
	var isDirty bool

	commitID, err := gitRepo.CommitID()
	if err != nil {
		return -1, err
	}

	isDirty, err = gitRepo.WorktreeIsDirty()
	if err != nil {
		return -1, err
	}

	var result storage.Result
	if runResult.Result.ExitCode == 0 {
		result = storage.ResultSuccess
	} else {
		result = storage.ResultFailure
	}

	totalDigest, err := inputs.Digest()
	if err != nil {
		return -1, err
	}

	storageInputs, err := inputsToStorageInputs(inputs)
	if err != nil {
		return -1, err
	}

	storageOutputs, err := toStorageOutputs(uploads)
	if err != nil {
		return -1, err
	}

	tr := storage.TaskRunFull{
		TaskRun: storage.TaskRun{
			ApplicationName:  task.AppName,
			TaskName:         task.Name,
			VCSRevision:      commitID,
			VCSIsDirty:       isDirty,
			StartTimestamp:   runResult.StartTime,
			StopTimestamp:    runResult.StopTime,
			TotalInputDigest: totalDigest.String(),
			Result:           result,
		},
		Inputs:  *storageInputs,
		Outputs: storageOutputs,
	}

	return storer.SaveTaskRun(ctx, &tr)
}

func inputsToStorageInputs(inputs *Inputs) (*storage.Inputs, error) {
	var result storage.Inputs

	for _, in := range inputs.Inputs() {
		digest, err := in.Digest()
		if err != nil {
			return nil, fmt.Errorf("calculating digest of %q failed: %w", in, err)
		}

		switch v := in.(type) {
		case *InputFile:
			result.Files = append(result.Files, &storage.InputFile{
				Path:   v.RelPath(),
				Digest: digest.String(),
			})

		case *InputString:
			result.Strings = append(result.Strings, &storage.InputString{
				String: v.Value(),
				Digest: digest.String(),
			})

		case *InputEnvVar:
			result.EnvironmentVariables = append(result.EnvironmentVariables, &storage.InputEnvVar{
				Name:   v.Name(),
				Digest: digest.String(),
			})
		case *InputTask:
			result.TaskInfo = append(result.TaskInfo, &storage.InputTaskInfo{
				Name:   v.TaskID(),
				Digest: digest.String(),
			})
		default:
			return nil, fmt.Errorf("unsupported input type %T", v)

		}
	}

	return &result, nil
}

func toStorageOutputs(uploadResults []*UploadResult) ([]*storage.Output, error) {
	resultMap := make(map[Output]*storage.Output)

	for _, uploadResult := range uploadResults {
		output, exist := resultMap[uploadResult.Output]
		if !exist {
			size, err := uploadResult.Output.SizeBytes()
			if err != nil {
				return nil, fmt.Errorf("getting size of %q failed: %w", uploadResult.Output, err)
			}

			digest, err := uploadResult.Output.Digest()
			if err != nil {
				return nil, fmt.Errorf("calculating digest of %q failed: %w", uploadResult.Output, err)
			}

			var outputType storage.ArtifactType
			switch uploadResult.Output.Type() {
			case DockerOutput:
				outputType = storage.ArtifactTypeDocker
			case FileOutput:
				outputType = storage.ArtifactTypeFile
			default:
				return nil, fmt.Errorf("output %q is of unsupported type: %s", uploadResult.Output, uploadResult.Output.Type())
			}

			output = &storage.Output{
				Name:      uploadResult.Output.Name(),
				Type:      outputType,
				Digest:    digest.String(),
				SizeBytes: size,
			}

			resultMap[uploadResult.Output] = output
		}

		var method storage.UploadMethod
		switch uploadResult.Method {
		case UploadMethodDocker:
			method = storage.UploadMethodDockerRegistry
		case UploadMethodFilecopy:
			method = storage.UploadMethodFileCopy
		case UploadMethodS3:
			method = storage.UploadMethodS3
		}

		output.Uploads = append(output.Uploads,
			&storage.Upload{
				URI:                  uploadResult.URL,
				UploadStartTimestamp: uploadResult.Start,
				UploadStopTimestamp:  uploadResult.Stop,
				Method:               method,
			})
	}

	return outputMapToSlice(resultMap), nil
}

func outputMapToSlice(m map[Output]*storage.Output) []*storage.Output {
	result := make([]*storage.Output, 0, len(m))

	for _, v := range m {
		result = append(result, v)
	}

	return result
}
