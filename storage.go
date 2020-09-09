package baur

import (
	"context"
	"errors"
	"fmt"

	"github.com/simplesurance/baur/v1/internal/vcs"
	"github.com/simplesurance/baur/v1/storage"
)

func StoreRun(
	ctx context.Context,
	storer storage.Storer,
	vcsState vcs.StateFetcher,
	task *Task,
	inputs *Inputs,
	runResult *RunResult,
	uploads []*UploadResult,
) (int, error) {
	var commitID string
	var isDirty bool

	commitID, err := vcsState.CommitID()
	if err != nil && !errors.Is(err, vcs.ErrVCSRepositoryNotExist) {
		return -1, err
	}

	isDirty, err = vcsState.WorktreeIsDirty()
	if err != nil && !errors.Is(err, vcs.ErrVCSRepositoryNotExist) {
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

	storageInputs, err := InputsToStorageInputs(inputs)
	if err != nil {
		return -1, err
	}

	storageOutputs, err := ToStorageOutputs(uploads)
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
		Inputs:  storageInputs,
		Outputs: storageOutputs,
	}

	return storer.SaveTaskRun(ctx, &tr)
}

func InputsToStorageInputs(inputs *Inputs) ([]*storage.Input, error) {
	result := make([]*storage.Input, 0, len(inputs.Files))

	for _, file := range inputs.Files {
		// TODO: rename storage.Input.URI to Name?
		digest, err := file.Digest()
		if err != nil {
			return nil, fmt.Errorf("calculating digest of %q failed: %w", file.Path(), err)
		}

		result = append(result, &storage.Input{
			URI:    file.RepoRelPath(),
			Digest: digest.String(),
		})
	}

	if inputs.AdditionalStr.Exists() {
		digest, err := inputs.AdditionalStr.Digest()
		if err != nil {
			return nil, fmt.Errorf("calculating digest for additional string %q failed: %w", inputs.AdditionalStr.Value, err)
		}
		result = append(result, &storage.Input{
			URI:    inputs.AdditionalStr.String(),
			Digest: digest.String(),
		})
	}

	return result, nil
}

func ToStorageOutputs(uploadResults []*UploadResult) ([]*storage.Output, error) {
	resultMap := make(map[Output]*storage.Output)

	for _, uploadResult := range uploadResults {
		output, exist := resultMap[uploadResult.Output]
		if !exist {
			size, err := uploadResult.Output.Size()
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
