package baur

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/simplesurance/baur/log"
	"github.com/simplesurance/baur/storage"
)

const (
	_ Status = iota
	// BuildStatusInputsUndefined inputs of the application are undefined,
	BuildStatusInputsUndefined
	// BuildStatusExist a build exist
	BuildStatusExist
	// BuildStatusOutstanding no build exist
	BuildStatusOutstanding
)

// Status indicates if build for a current application version exist
type Status int

func (b Status) IsNotNull() bool {
	return b > 0
}

func (b Status) String() string {
	switch b {
	case BuildStatusInputsUndefined:
		return "Inputs Undefined"
	case BuildStatusExist:
		return "Exist"
	case BuildStatusOutstanding:
		return "Outstanding"
	default:
		panic(fmt.Sprintf("incompatible Status value: %d", b))
	}
}

func NewAppStatus(s string) (Status, error) {
	switch s {
	case "inputs_undefined":
		return BuildStatusInputsUndefined, nil
	case "exist":
		return BuildStatusExist, nil
	case "outstanding":
		return BuildStatusOutstanding, nil
	default:
		return -1,
			fmt.Errorf("invalid status \"%s\". Valid choices are inputs_undefined, exist or outstanding", s)
	}
}

// GetAppStatus calculates the total input digest of the app and checks in the
// storage if a build for this input digest already exist.
// If the function returns BuildStatusExist the returned build pointer is valid
// otherwise it is nil.
func GetAppStatus(storer storage.Storer, app *App) (Status, *storage.Build, error) {
	if len(app.BuildInputPaths) == 0 {
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

		log.Fatalf("fetching build of %q from storage failed: %s\n", app.Name, err)
	}

	return BuildStatusExist, build, nil
}
