package cfg

import (
	"errors"

	"github.com/simplesurance/baur/v3/internal/deepcopy"
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

// validate checks if the stored information is valid.
func (out *OutputInclude) validate() error {
	if err := validateIncludeID(out.IncludeID); err != nil {
		if out.IncludeID != "" {
			err = fieldErrorWrap(err, out.IncludeID)
		}
		return err
	}

	if len(out.DockerImage) == 0 && len(out.File) == 0 {
		return errors.New("no output is defined")
	}

	return outputValidate(out)
}

func (out *OutputInclude) clone() *OutputInclude {
	var clone OutputInclude

	deepcopy.MustCopy(out, &clone)
	clone.filepath = out.filepath

	return &clone
}
