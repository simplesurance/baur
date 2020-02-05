package cfg

import (
	"github.com/simplesurance/baur/cfg/resolver"
)

// Output is the tasks output section
type Output struct {
	DockerImage []*DockerImageOutput `comment:"Docker images that are produced by the [Task.command]"`
	File        []*FileOutput        `comment:"Files that are produces by the [Task.command]"`
}

func (out *Output) DockerImageOutputs() *[]*DockerImageOutput {
	return &out.DockerImage
}

func (out *Output) FileOutputs() *[]*FileOutput {
	return &out.File
}

func (out *Output) Resolve(resolvers resolver.Resolver) error {
	for _, dockerImage := range out.DockerImage {
		if err := dockerImage.Resolve(resolvers); err != nil {
			return FieldErrorWrap(err, "DockerImage")
		}
	}

	for _, file := range out.File {
		if err := file.Resolve(resolvers); err != nil {
			return FieldErrorWrap(err, "FileOutput")
		}
	}

	return nil
}

// Merge appends the information in other to out.
func (out *Output) Merge(other OutputDef) {
	out.DockerImage = append(out.DockerImage, *other.DockerImageOutputs()...)
	out.File = append(out.File, *other.FileOutputs()...)
}

// Validate checks that the stored information is valid.
func OutputValidate(o OutputDef) error {
	for _, f := range *o.FileOutputs() {
		if err := f.Validate(); err != nil {
			return FieldErrorWrap(err, "File")
		}
	}

	for _, d := range *o.DockerImageOutputs() {
		if err := d.Validate(); err != nil {
			return FieldErrorWrap(err, "DockerImage")
		}
	}

	return nil
}
