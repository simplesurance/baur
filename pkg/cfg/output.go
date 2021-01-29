package cfg

import (
	"github.com/simplesurance/baur/v1/pkg/cfg/resolver"
)

// Output is the tasks output section
type Output struct {
	DockerImage []DockerImageOutput `comment:"Docker images that are produced by the [Task.command]"`
	File        []FileOutput        `comment:"Files that are produces by the [Task.command]"`
}

func (out *Output) DockerImageOutputs() []DockerImageOutput {
	return out.DockerImage
}

func (out *Output) FileOutputs() []FileOutput {
	return out.File
}

func (out *Output) Resolve(resolvers resolver.Resolver) error {
	for i, dockerImage := range out.DockerImage {
		if err := dockerImage.Resolve(resolvers); err != nil {
			return fieldErrorWrap(err, "DockerImage")
		}

		// replace the slice element because dockerImage is a copy
		out.DockerImage[i] = dockerImage
	}

	for i, file := range out.File {
		if err := file.resolve(resolvers); err != nil {
			return fieldErrorWrap(err, "FileOutput")
		}

		// replace the slice element because file is a copy
		out.File[i] = file
	}

	return nil
}

// Merge appends the information in other to out.
func (out *Output) Merge(other OutputDef) {
	out.DockerImage = append(out.DockerImage, other.DockerImageOutputs()...)
	out.File = append(out.File, other.FileOutputs()...)
}

func outputValidate(o OutputDef) error {
	for _, f := range o.FileOutputs() {
		if err := f.validate(); err != nil {
			return fieldErrorWrap(err, "File")
		}
	}

	for _, d := range o.DockerImageOutputs() {
		if err := d.validate(); err != nil {
			return fieldErrorWrap(err, "DockerImage")
		}
	}

	return nil
}
