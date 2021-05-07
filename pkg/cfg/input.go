package cfg

// Input contains information about task inputs
type Input struct {
	Files         []FileInputs
	GolangSources []GolangSources `comment:"Inputs specified by resolving dependencies of Golang source files or packages."`
}

func (in *Input) IsEmpty() bool {
	return len(in.Files) == 0 && len(in.GolangSources) == 0
}

func (in *Input) fileInputs() []FileInputs {
	return in.Files
}

func (in *Input) golangSourcesInputs() []GolangSources {
	return in.GolangSources
}

// merge appends the information in other to in.
func (in *Input) merge(other inputDef) {
	in.Files = append(in.Files, other.fileInputs()...)
	in.GolangSources = append(in.GolangSources, other.golangSourcesInputs()...)
}

func (in *Input) resolve(resolver Resolver) error {
	for _, f := range in.Files {
		if err := f.resolve(resolver); err != nil {
			return fieldErrorWrap(err, "Files")
		}
	}

	for i, gs := range in.GolangSources {
		if err := gs.resolve(resolver); err != nil {
			return fieldErrorWrap(err, "GoLangSources")
		}

		in.GolangSources[i] = gs
	}

	return nil
}

// inputValidate validates the Input section
func inputValidate(i inputDef) error {
	for _, f := range i.fileInputs() {
		if err := f.validate(); err != nil {
			return fieldErrorWrap(err, "Files")
		}
	}

	for _, gs := range i.golangSourcesInputs() {
		if err := gs.validate(); err != nil {
			return fieldErrorWrap(err, "GolangSources")
		}
	}

	return nil
}
