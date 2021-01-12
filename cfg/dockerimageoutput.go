package cfg

import (
	"github.com/simplesurance/baur/v1/cfg/resolver"
)

//TODO: make RegistryUpload a pointer so we do not have to check if its empty

// DockerImageOutput describes where a docker container is uploaded to.
type DockerImageOutput struct {
	IDFile         string                      `toml:"idfile" comment:"Path to a file that is created by the [Task.Command] and contains the image ID of the produced image (docker build --iidfile)."`
	RegistryUpload []DockerImageRegistryUpload `comment:"Registry and repository the image is uploaded to"`
}

func (d *DockerImageOutput) Resolve(resolvers resolver.Resolver) error {
	var err error

	if d.IDFile, err = resolvers.Resolve(d.IDFile); err != nil {
		return fieldErrorWrap(err, "idfile")
	}

	for i, upload := range d.RegistryUpload {
		if err = upload.Resolve(resolvers); err != nil {
			return fieldErrorWrap(err, "RegistryUpload")
		}

		d.RegistryUpload[i] = upload
	}

	return nil
}

// validate validates its content
func (d *DockerImageOutput) validate() error {
	if len(d.IDFile) == 0 {
		return newFieldError("can not be empty", "idfile")
	}

	for _, upload := range d.RegistryUpload {
		if err := upload.validate(); err != nil {
			return fieldErrorWrap(err, "RegistryUpload")
		}
	}

	return nil
}
