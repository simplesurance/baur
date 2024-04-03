package cfg

// Input contains information about task inputs
type Input struct {
	EnvironmentVariables []EnvVarsInputs
	Files                []FileInputs
	GolangSources        []GolangSources `comment:"Inputs specified by resolving dependencies of Golang source files or packages."`
	TaskInfos            []TaskInfo      `comment:"Information about another baur task."`
	ExcludedFiles        FileExcludeList
}

func (in *Input) IsEmpty() bool {
	return len(in.Files) == 0 &&
		len(in.GolangSources) == 0 &&
		len(in.EnvironmentVariables) == 0 &&
		len(in.ExcludedFiles.Paths) == 0 &&
		len(in.TaskInfos) == 0
}

func (in *Input) fileInputs() []FileInputs {
	return in.Files
}

func (in *Input) golangSourcesInputs() []GolangSources {
	return in.GolangSources
}

func (in *Input) envVariables() []EnvVarsInputs {
	return in.EnvironmentVariables
}

func (in *Input) excludedFiles() *FileExcludeList {
	return &in.ExcludedFiles
}

func (in *Input) taskInfos() []TaskInfo {
	return in.TaskInfos
}

// merge appends the information in other to in.
func (in *Input) merge(other inputDef) {
	in.Files = append(in.Files, other.fileInputs()...)
	in.GolangSources = append(in.GolangSources, other.golangSourcesInputs()...)
	in.EnvironmentVariables = append(in.EnvironmentVariables, other.envVariables()...)
	in.ExcludedFiles.Paths = append(in.ExcludedFiles.Paths, other.excludedFiles().Paths...)
	in.TaskInfos = append(in.TaskInfos, other.taskInfos()...)
}

func (in *Input) resolve(resolver Resolver) error {
	for _, f := range in.Files {
		if err := f.resolve(resolver); err != nil {
			return fieldErrorWrap(err, "Files")
		}
	}

	if err := in.ExcludedFiles.resolve(resolver); err != nil {
		return fieldErrorWrap(err, "ExcludedFiles")
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

	for _, env := range i.envVariables() {
		if err := env.Validate(); err != nil {
			return fieldErrorWrap(err, "EnvVariables")
		}
	}

	if err := i.excludedFiles().Validate(); err != nil {
		return fieldErrorWrap(err, "ExcludedFiles")
	}

	if err := validateTaskInfos(i.taskInfos()); err != nil {
		return fieldErrorWrap(err, "TaskInfos")
	}

	return nil
}
