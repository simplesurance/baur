package cfg

type inputDef interface {
	envVariables() []EnvVarsInputs
	fileInputs() []FileInputs
	golangSourcesInputs() []GolangSources
	excludedFiles() *FileExcludeList
	taskInfos() []TaskInfo
}
