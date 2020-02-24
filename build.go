package baur

import (
	"fmt"

	"github.com/simplesurance/baur/storage"
)

// BuildStatus indicates if build for a current application version exist
type BuildStatus int

const (
	_ BuildStatus = iota
	BuildStatusUndefined
	BuildStatusInputsUndefined
	// TODO: is BuildStatusBuildCommandUndefined used/needed?
	BuildStatusExist
	BuildStatusPending
)

func (t BuildStatus) String() string {
	switch t {
	case BuildStatusUndefined:
		return "undefined"
	case BuildStatusInputsUndefined:
		return "Inputs Undefined"
	case BuildStatusExist:
		return "Exist"
	case BuildStatusPending:
		return "Pending"
	default:
		panic(fmt.Sprintf("incompatible TaskStatus value: %d", t))
	}
}

// TaskRunStatus resolves the file inputs of the task, calculates the total input digestand checks in the
// store if a run for this input digest already exist.
// If the function returns BuildStatusExist the returned build pointer is valid
// otherwise it is nil.
func TaskRunStatus(task *Task, repositoryDir string, store storage.Storer) (BuildStatus, *storage.BuildWithDuration, error) {
	if !task.HasInputs() {
		return BuildStatusInputsUndefined, nil, nil
	}

	resolver := NewInputResolver()

	inputs, err := resolver.Resolve(repositoryDir, task)
	if err != nil {
		return BuildStatusUndefined, nil, err
	}

	return taskStatusFromDB(task.AppName, inputs, store)
}

func TaskRunStatusInputs(task *Task, inputs *Inputs, store storage.Storer) (BuildStatus, *storage.BuildWithDuration, error) {
	if !task.HasInputs() {
		return BuildStatusInputsUndefined, nil, nil
	}

	return taskStatusFromDB(task.AppName, inputs, store)
}

func taskStatusFromDB(appName string, inputs *Inputs, store storage.Storer) (BuildStatus, *storage.BuildWithDuration, error) {
	digest, err := inputs.Digest()
	if err != nil {
		return BuildStatusUndefined, nil, err
	}

	existingBuild, err := store.GetLatestBuildByDigest(appName, digest.String())
	if err != nil {
		if err == storage.ErrNotExist {
			return BuildStatusPending, nil, nil
		}

		return BuildStatusUndefined, nil, err
	}

	return BuildStatusExist, existingBuild, nil
}
