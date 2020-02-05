package cfg

import (
	"github.com/simplesurance/baur/cfg/resolver"
)

//TODO: make RegistryUpload a pointer so we do not have to check if its empty

// DockerImageOutput describes where a docker container is uploaded to.
type DockerImageOutput struct {
	IDFile         string                    `toml:"idfile" comment:"Path to a file that is created by the [Task.Command] and contains the image ID of the produced image (docker build --iidfile).\n Valid variables: $ROOT, $APPNAME" commented:"true"`
	RegistryUpload DockerImageRegistryUpload `comment:"Registry and repository the image is uploaded to"`
}

func (d *DockerImageOutput) Resolve(resolvers resolver.Resolver) error {
	var err error

	if d.IDFile, err = resolvers.Resolve(d.IDFile); err != nil {
		return FieldErrorWrap(err, "idfile")
	}

	if err = d.RegistryUpload.Resolve(resolvers); err != nil {
		return FieldErrorWrap(err, "RegistryUpload")
	}

	return nil
}

// Validate validates its content
func (d *DockerImageOutput) Validate() error {
	if len(d.IDFile) == 0 {
		return NewFieldError("can not be empty", "idfile")
	}

	if err := d.RegistryUpload.Validate(); err != nil {
		return FieldErrorWrap(err, "RegistryUpload")
	}

	return nil
}

// IsEmpty returns true if the object contains no data.
func (d *DockerImageOutput) IsEmpty() bool {
	return d.IDFile == "" && d.RegistryUpload.IsEmpty()
}
