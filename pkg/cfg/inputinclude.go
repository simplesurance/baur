package cfg

import (
	"github.com/simplesurance/baur/v1/internal/deepcopy"
)

// InputInclude is a reusable Input definition.
type InputInclude struct {
	IncludeID string `toml:"include_id" comment:"identifier of the include"`

	Files         []FileInputs    `comment:"Inputs specified by file glob paths"`
	GolangSources []GolangSources `comment:"Inputs specified by directories containing Golang applications"`

	filepath string
}

func (in *InputInclude) FileInputs() []FileInputs {
	return in.Files
}

func (in *InputInclude) GolangSourcesInputs() []GolangSources {
	return in.GolangSources
}

// validate checks if the stored information is valid.
func (in *InputInclude) validate() error {
	if err := validateIncludeID(in.IncludeID); err != nil {
		if in.IncludeID != "" {
			err = fieldErrorWrap(err, in.IncludeID)
		}
		return err
	}

	if InputsAreEmpty(in) {
		return nil
	}

	if err := inputValidate(in); err != nil {
		return err
	}

	return nil
}

func (in *InputInclude) clone() *InputInclude {
	var clone InputInclude

	deepcopy.MustCopy(in, &clone)
	// filepath is assigned manually because filepath is a private field, MustCopy() only clones exported fields
	clone.filepath = in.filepath

	return &clone
}
