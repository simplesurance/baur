package cfg

import "github.com/simplesurance/baur/v1/cfg/resolver"

// Input contains information about task inputs
type Input struct {
	Files         FileInputs         `comment:"Inputs specified by file glob paths"`
	GitFiles      GitFileInputs      `comment:"Inputs specified by path, matching only Git tracked files"`
	GolangSources GolangSources      `comment:"Inputs specified by resolving dependencies of Golang source files or packages."`
	AdditionalStr AdditionalInputStr `comment:"Additional input specified by command line"`
}

func (in *Input) FileInputs() *FileInputs {
	return &in.Files
}

func (in *Input) GitFileInputs() *GitFileInputs {
	return &in.GitFiles
}

func (in *Input) GolangSourcesInputs() *GolangSources {
	return &in.GolangSources
}

func (in *Input) AdditionalStrInput() *AdditionalInputStr {
	return &in.AdditionalStr
}

// Merge appends the information in other to in.
func (in *Input) Merge(other InputDef) {
	in.Files.Merge(other.FileInputs())
	in.GitFiles.Merge(other.GitFileInputs())
	in.GolangSources.Merge(other.GolangSourcesInputs())
	in.AdditionalStr = *other.AdditionalStrInput()
}

func (in *Input) Resolve(resolvers resolver.Resolver) error {
	if err := in.Files.Resolve(resolvers); err != nil {
		return FieldErrorWrap(err, "Files")
	}

	if err := in.GitFiles.Resolve(resolvers); err != nil {
		return FieldErrorWrap(err, "Gitfiles")
	}

	if err := in.GolangSources.Resolve(resolvers); err != nil {
		return FieldErrorWrap(err, "GoLangSources")
	}

	if err := in.AdditionalStr.Resolve(resolvers); err != nil {
		return FieldErrorWrap(err, "AdditionalStr")
	}

	return nil
}

// InputValidate validates the Input section
func InputValidate(i InputDef) error {
	if err := i.FileInputs().Validate(); err != nil {
		return FieldErrorWrap(err, "Files")
	}

	if err := i.GolangSourcesInputs().Validate(); err != nil {
		return FieldErrorWrap(err, "GolangSources")
	}

	// TODO: add validation for gitfiles section

	return nil
}
