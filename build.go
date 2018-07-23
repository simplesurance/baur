package baur

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/simplesurance/baur/log"
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
// If a build exist for the totalInputDigest it returns the ID of the build that
// was build last.
func GetBuildStatus(storer storage.Storer, app *App) (BuildStatus, int64, error) {
	if len(app.BuildInputPaths) == 0 {
		return BuildStatusInputsUndefined, -1, nil
	}

	d, err := app.TotalInputDigest()
	if err != nil {
		return -1, -1, errors.Wrap(err, "calculating total input digest failed")
	}

	buildID, err := storer.FindLatestAppBuildByDigest(app.Name, d.String())
	if err != nil {
		if err == storage.ErrNotExist {
			return BuildStatusOutstanding, -1, nil
		}

		log.Fatalf("fetching build of %q from storage failed: %s\n", app.Name, err)
	}

	return BuildStatusExist, buildID, nil
}
