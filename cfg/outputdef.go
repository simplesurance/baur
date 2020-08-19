package cfg

type OutputDef interface {
	DockerImageOutputs() []DockerImageOutput
	FileOutputs() []FileOutput
}
