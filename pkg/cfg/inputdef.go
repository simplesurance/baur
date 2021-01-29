package cfg

type InputDef interface {
	FileInputs() []FileInputs
	GolangSourcesInputs() []GolangSources
}

// InputsAreEmpty returns true if no inputs are defined
func InputsAreEmpty(in InputDef) bool {
	return len(in.FileInputs()) == 0 &&
		len(in.GolangSourcesInputs()) == 0
}
