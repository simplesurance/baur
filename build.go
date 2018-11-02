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
	// BuildStatusExist a build exist
	BuildStatusExist
	// BuildStatusOutstanding no build exist
	BuildStatusOutstanding
)

func (b BuildStatus) String() string {
	switch b {
	case BuildStatusInputsUndefined:
		return "Inputs Undefined"
	case BuildStatusExist:
		return "Exist"
	case BuildStatusOutstanding:
		return "Outstanding"
	default:
		panic(fmt.Sprintf("incompatible BuildStatus value: %d", b))
	}
}

// GetBuildStatus calculates the total input digest of the app and checks in the
// storage if a build for this input digest already exist.
// If the function returns BuildStatusExist the returned build pointer is valid
// otherwise it is nil.
func GetBuildStatus(storer storage.Storer, app *App) (BuildStatus, *storage.BuildWithDuration, error) {
	if !app.HasBuildInputs() {
		return BuildStatusInputsUndefined, nil, nil
	}

	d, err := app.TotalInputDigest()
	if err != nil {
		return -1, nil, errors.Wrap(err, "calculating total input digest failed")
	}

	build, err := storer.GetLatestBuildByDigest(app.Name, d.String())
	if err != nil {
		if err == storage.ErrNotExist {
			return BuildStatusOutstanding, nil, nil
		}

		return -1, nil, errors.Wrap(err, "fetching latest build failed")
	}

	return BuildStatusExist, build, nil
}
