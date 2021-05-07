package cfg

type inputDef interface {
	fileInputs() []FileInputs
	golangSourcesInputs() []GolangSources
}
