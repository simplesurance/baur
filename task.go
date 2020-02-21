package baur

import (
	"fmt"
	"path"
	"path/filepath"
	"sort"

	"github.com/simplesurance/baur/cfg"
	"github.com/simplesurance/baur/upload/scheduler"
)

// Task is a an execution step belonging to an app.
// A task has a set of Inputs that produce a set of outputs by executing it's
// Command.
type Task struct {
	RepositoryRoot string
	Directory      string

	AppName string

	Name             string
	Command          string
	UnresolvedInputs *cfg.Input
	Outputs          *cfg.Output
}

// NewTask returns a new Task.
func NewTask(cfg *cfg.Task, appName, repositoryRootdir, workingDir string) *Task {
	return &Task{
		RepositoryRoot:   repositoryRootdir,
		Directory:        workingDir,
		Outputs:          &cfg.Output,
		Command:          cfg.Command,
		Name:             cfg.Name,
		AppName:          appName,
		UnresolvedInputs: &cfg.Input,
	}
}

// ID returns <APP-NAME>.<TASK-NAME>
func (t *Task) ID() string {
	return fmt.Sprintf("%s.%s", t.AppName, t.Name)
}

// String returns ID()
func (t *Task) String() string {
	return t.ID()
}

// HasInputs returns true if Inputs are defined for the app
func (t *Task) HasInputs() bool {
	return !cfg.InputsAreEmpty(t.UnresolvedInputs)
}

// TODO: rename this function when the db and commands support multiple tasks.
// BuildOutputs returns a list of outputs that the task.Command is expected to produce.
func (t *Task) BuildOutputs() ([]BuildOutput, error) {
	result := make([]BuildOutput, 0, len(t.Outputs.DockerImage)+len(t.Outputs.File))

	taskRelDir, err := filepath.Rel(t.Directory, t.RepositoryRoot)
	if err != nil {
		return nil, err
	}

	for _, di := range t.Outputs.DockerImage {
		result = append(result, &DockerArtifact{
			ImageIDFile: path.Join(t.Directory, di.IDFile),
			Tag:         di.RegistryUpload.Tag,
			Repository:  di.RegistryUpload.Repository,
			Registry:    di.RegistryUpload.Registry,
		})
	}

	for _, f := range t.Outputs.File {
		filePath := f.Path
		absFilePath := path.Join(t.Directory, filePath)

		if !f.S3Upload.IsEmpty() {
			url := "s3://" + f.S3Upload.Bucket + "/" + f.S3Upload.DestFile

			result = append(result, &FileArtifact{
				RelPath:   path.Join(taskRelDir, filePath),
				Path:      absFilePath,
				DestFile:  f.S3Upload.DestFile,
				UploadURL: url,
				uploadJob: &scheduler.S3Job{
					DestURL:  url,
					FilePath: absFilePath,
				},
			})
		}

		if !f.FileCopy.IsEmpty() {
			result = append(result, &FileArtifact{
				RelPath:   path.Join(taskRelDir, filePath),
				Path:      absFilePath,
				DestFile:  f.FileCopy.Path,
				UploadURL: f.FileCopy.Path,
				uploadJob: &scheduler.FileCopyJob{
					Src: absFilePath,
					Dst: f.FileCopy.Path,
				},
			})
		}
	}

	return result, nil
}

func SortTasksByID(tasks []*Task) {
	sort.Slice(tasks, func(i int, j int) bool {
		return tasks[i].ID() < tasks[i].ID()
	})
}
