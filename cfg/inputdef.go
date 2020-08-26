package cfg

type InputDef interface {
	FileInputs() *FileInputs
	GitFileInputs() *GitFileInputs
	GolangSourcesInputs() *GolangSources
}

// TODO: make FileInputs, GitFileInputs, GolangSources pointers and get rid of this function?
// InputsAreEmpty returns true if no inputs are defined
func InputsAreEmpty(in InputDef) bool {
	return len(in.FileInputs().Paths) == 0 &&
		len(in.GitFileInputs().Paths) == 0 &&
		len(in.GolangSourcesInputs().Queries) == 0 &&
		len(in.GolangSourcesInputs().Environment) == 0
}
