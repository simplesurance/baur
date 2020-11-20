package cfg

import (
	"github.com/simplesurance/baur/v1/cfg/resolver"
)

// DockerImageRegistryUpload stores information about a Docker image upload.
type DockerImageRegistryUpload struct {
	Registry   string `toml:"registry" comment:"Registry address in the format <HOST>:[<PORT>]. If it's empty the default from the docker agent is used."`
	Repository string `toml:"repository" comment:"Repository name, Valid variables: $APPNAME"`
	Tag        string `toml:"tag" comment:"Tag that is applied to the image.\n Valid variables: $APPNAME, $UUID, $GITCOMMIT"`
}

func (d *DockerImageRegistryUpload) Resolve(resolvers resolver.Resolver) error {
	var err error

	if d.Repository, err = resolvers.Resolve(d.Repository); err != nil {
		return fieldErrorWrap(err, "repository")
	}

	if d.Tag, err = resolvers.Resolve(d.Tag); err != nil {
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
