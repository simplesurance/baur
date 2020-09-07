package cfg

type InputDef interface {
	FileInputs() *FileInputs
	GitFileInputs() *GitFileInputs
	GolangSourcesInputs() *GolangSources
	AdditionalStrInput() *AdditionalInputStr
}

// InputsAreEmpty returns true if no inputs are defined
func InputsAreEmpty(in InputDef) bool {
	return len(in.FileInputs().Paths) == 0 &&
		len(in.GitFileInputs().Paths) == 0 &&
		in.GolangSourcesInputs().IsEmpty() &&
		len(in.AdditionalStrInput().Value) == 0
}
