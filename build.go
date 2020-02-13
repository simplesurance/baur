package baur

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/simplesurance/baur/storage"
)

// BuildStatus indicates if build for a current application version exist
type BuildStatus int

const (
	_ BuildStatus = iota
	// BuildStatusInputsUndefined inputs of the application are undefined,
	BuildStatusInputsUndefined
	// BuildStatusBuildCommandUndefined build.command of the application is undefined,
	BuildStatusBuildCommandUndefined
	// BuildStatusExist a build exist
	BuildStatusExist
	// BuildStatusPending no build exist
	BuildStatusPending
)

func (b BuildStatus) String() string {
	switch b {
	case BuildStatusInputsUndefined:
		return "Inputs Undefined"
	case BuildStatusExist:
		return "Exist"
	case BuildStatusPending:
		return "Pending"
	case BuildStatusBuildCommandUndefined:
		return "Build Command Undefined"
	default:
		panic(fmt.Sprintf("incompatible BuildStatus value: %d", b))
	}
}

// GetBuildStatus calculates the total input digest of the app and checks in the
// storage if a build for this input digest already exist.
// If the function returns BuildStatusExist the returned build pointer is valid
// otherwise it is nil.
func GetBuildStatus(storer storage.Storer, task *Task) (BuildStatus, *storage.BuildWithDuration, error) {
	if len(task.Command) == 0 {
		return BuildStatusBuildCommandUndefined, nil, nil
	}

	if !task.HasInputs() {
		return BuildStatusInputsUndefined, nil, nil
	}

	d, err := task.TotalInputDigest()
	if err != nil {
		return -1, nil, errors.Wrap(err, "calculating total input digest failed")
	}

	build, err := storer.GetLatestBuildByDigest(task.AppName, d.String())
	if err != nil {
		if err == storage.ErrNotExist {
			return BuildStatusPending, nil, nil
		}

		return -1, nil, errors.Wrap(err, "fetching latest build failed")
	}

	return BuildStatusExist, build, nil
}
