package cfg

import (
	"errors"

	"github.com/simplesurance/baur/v1/internal/deepcopy"
)

// OutputInclude is a reusable Output definition
type OutputInclude struct {
	IncludeID string `toml:"include_id" comment:"identifier of the include"`

	DockerImage []DockerImageOutput `comment:"Docker images that are produced by the [Task.command]"`
	File        []FileOutput        `comment:"Files that are produces by the [Task.command]"`

	filepath string
}

func (out *OutputInclude) DockerImageOutputs() []DockerImageOutput {
	return out.DockerImage
}

func (out *OutputInclude) FileOutputs() []FileOutput {
	return out.File
}

// Validate checks if the stored information is valid.
func (out *OutputInclude) Validate() error {
	if err := validateIncludeID(out.IncludeID); err != nil {
		if out.IncludeID != "" {
			err = FieldErrorWrap(err, out.IncludeID)
		}
		return err
	}

	if len(out.DockerImage) == 0 && len(out.File) == 0 {
		return errors.New("no output is defined")
	}

	if err := OutputValidate(out); err != nil {
		return err
	}

	return nil
}

func (out *OutputInclude) clone() *OutputInclude {
	var clone OutputInclude

	deepcopy.MustCopy(out, &clone)
	clone.filepath = out.filepath

	return &clone
}
