package cfg

import (
	"github.com/simplesurance/baur/cfg/resolver"
)

// DockerImageRegistryUpload stores information about a Docker image upload.
type DockerImageRegistryUpload struct {
	Registry   string `toml:"registry" comment:"Registry address in the format <HOST>:[<PORT>]. If it's empty the default from the docker agent is used." commented:"true"`
	Repository string `toml:"repository" comment:"Repository name, Valid variables: $APPNAME" commented:"true"`
	Tag        string `toml:"tag" comment:"Tag that is applied to the image.\n Valid variables: $APPNAME, $UUID, $GITCOMMIT" commented:"true"`
}

//IsEmpty returns true if the struct contains no information.
func (d *DockerImageRegistryUpload) IsEmpty() bool {
	return len(d.Registry) == 0 && len(d.Repository) == 0 && len(d.Tag) == 0
}

func (d *DockerImageRegistryUpload) Resolve(resolvers resolver.Resolver) error {
	var err error

	if d.Repository, err = resolvers.Resolve(d.Repository); err != nil {
		return FieldErrorWrap(err, "repository")
	}

	if d.Tag, err = resolvers.Resolve(d.Tag); err != nil {
		return FieldErrorWrap(err, "tag")
	}

	return nil
}

func (d *DockerImageRegistryUpload) Validate() error {
	if len(d.Repository) == 0 {
		return NewFieldError("can not be empty", "repository")
	}

	if len(d.Tag) == 0 {
		return NewFieldError("can not be empty", "tag")
	}

	return nil
}
