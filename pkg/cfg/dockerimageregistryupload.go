package cfg

// DockerImageRegistryUpload stores information about a Docker image upload.
type DockerImageRegistryUpload struct {
	Registry   string `toml:"registry" comment:"Registry address in the format <HOST>:[<PORT>]. If it's empty the docker agent's default is used."`
	Repository string `toml:"repository"`
	Tag        string `toml:"tag"`
}

func (d *DockerImageRegistryUpload) Resolve(resolver Resolver) error {
	var err error

	if d.Registry, err = resolver.Resolve(d.Registry); err != nil {
		return fieldErrorWrap(err, "registry")
	}

	if d.Repository, err = resolver.Resolve(d.Repository); err != nil {
		return fieldErrorWrap(err, "repository")
	}

	if d.Tag, err = resolver.Resolve(d.Tag); err != nil {
		return fieldErrorWrap(err, "tag")
	}

	return nil
}

func (d *DockerImageRegistryUpload) validate() error {
	if len(d.Repository) == 0 {
		return newFieldError("can not be empty", "repository")
	}

	if len(d.Tag) == 0 {
		return newFieldError("can not be empty", "tag")
	}

	return nil
}
