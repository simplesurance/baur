package cfg

// InputInclude is a reusable Input definition.
type InputInclude struct {
	IncludeID string `toml:"include_id" comment:"identifier of the include"`

	Files         FileInputs         `comment:"Inputs specified by file glob paths"`
	GitFiles      GitFileInputs      `comment:"Inputs specified by path, matching only Git tracked files"`
	GolangSources GolangSources      `comment:"Inputs specified by directories containing Golang applications"`
	AdditionalStr AdditionalInputStr `comment:"Additional input specified by command line"`
}

func (in *InputInclude) FileInputs() *FileInputs {
	return &in.Files
}

func (in *InputInclude) GitFileInputs() *GitFileInputs {
	return &in.GitFiles
}

func (in *InputInclude) GolangSourcesInputs() *GolangSources {
	return &in.GolangSources
}

func (in *InputInclude) AdditionalStrInput() *AdditionalInputStr {
	return &in.AdditionalStr
}

// Validate checks if the stored information is valid.
func (in *InputInclude) Validate() error {
	if err := validateIncludeID(in.IncludeID); err != nil {
		if in.IncludeID != "" {
			err = FieldErrorWrap(err, in.IncludeID)
		}
		return err
	}

	if InputsAreEmpty(in) {
		return nil
	}

	if err := InputValidate(in); err != nil {
		return err
	}

	return nil
}
