package cfg

import (
	"github.com/simplesurance/baur/v1/cfg/resolver"
)

// Input contains information about task inputs
type Input struct {
	Files         []FileInputs    `comment:"Inputs specified by file glob paths"`
	GitFiles      []GitFileInputs `comment:"Inputs specified by path, matching only Git tracked files"`
	GolangSources []GolangSources `comment:"Inputs specified by resolving dependencies of Golang source files or packages."`
}

func (in *Input) FileInputs() []FileInputs {
	return in.Files
}

func (in *Input) GitFileInputs() []GitFileInputs {
	return in.GitFiles
}

func (in *Input) GolangSourcesInputs() []GolangSources {
	return in.GolangSources
}

// merge appends the information in other to in.
func (in *Input) merge(other InputDef) {
	in.Files = append(in.Files, other.FileInputs()...)
	in.GitFiles = append(in.GitFiles, other.GitFileInputs()...)
	in.GolangSources = append(in.GolangSources, other.GolangSourcesInputs()...)
}

func (in *Input) resolve(resolvers resolver.Resolver) error {
	for _, f := range in.Files {
		if err := f.resolve(resolvers); err != nil {
			return fieldErrorWrap(err, "Files")
		}
	}

	for _, g := range in.GitFiles {
		if err := g.resolve(resolvers); err != nil {
			return fieldErrorWrap(err, "Gitfiles")
		}
	}

	for i, gs := range in.GolangSources {
		if err := gs.resolve(resolvers); err != nil {
			return fieldErrorWrap(err, "GoLangSources")
		}

		// TODO is this needed? If not why not?
		in.GolangSources[i] = gs
	}

	return nil
}

// inputValidate validates the Input section
func inputValidate(i InputDef) error {
	for _, f := range i.FileInputs() {
		if err := f.validate(); err != nil {
			return fieldErrorWrap(err, "Files")
		}
	}

	for _, gs := range i.GolangSourcesInputs() {
		if err := gs.validate(); err != nil {
			return fieldErrorWrap(err, "GolangSources")
		}
	}

	// TODO: add validation for gitfiles section

	return nil
}
