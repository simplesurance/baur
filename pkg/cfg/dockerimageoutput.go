package cfg

// DockerImageOutput describes where a docker container is uploaded to.
type DockerImageOutput struct {
	IDFile         string `toml:"idfile" comment:"File containing the image ID of the produced image (docker build --iidfile)."`
	RegistryUpload []DockerImageRegistryUpload
}

func (d *DockerImageOutput) resolve(resolver Resolver) error {
	var err error

	if d.IDFile, err = resolver.Resolve(d.IDFile); err != nil {
		return fieldErrorWrap(err, "idfile")
	}

	for i, upload := range d.RegistryUpload {
		if err = upload.Resolve(resolver); err != nil {
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
