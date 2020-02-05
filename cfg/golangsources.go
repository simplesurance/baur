package cfg

import (
	"github.com/simplesurance/baur/cfg/resolver"
)

// GolangSources specifies inputs for Golang Applications
type GolangSources struct {
	Environment []string `toml:"environment" comment:"Environment to use when discovering Golang source files\n This can be environment variables understood by the Golang tools, like GOPATH, GOFLAGS, etc.\n If empty the default Go environment is used.\n Valid variables: $ROOT, $APPNAME" commented:"true"`
	Paths       []string `toml:"paths" comment:"Paths to directories containing Golang source files.\n All source files including imported packages are discovered,\n files from Go's stdlib package and testfiles are ignored. Valid variables: $ROOT, $APPNAME." commented:"true"`
}

// Merge merges the two GolangSources structs
func (g *GolangSources) Merge(other *GolangSources) {
	g.Paths = append(g.Paths, other.Paths...)
	g.Environment = append(g.Environment, other.Environment...)
}

func (g *GolangSources) Resolve(resolvers resolver.Resolver) error {
	for i, env := range g.Environment {
		var err error

		if g.Environment[i], err = resolvers.Resolve(env); err != nil {
			return FieldErrorWrap(err, "Environment", env)
		}
	}

	for i, p := range g.Paths {
		var err error

		if g.Paths[i], err = resolvers.Resolve(p); err != nil {
			return FieldErrorWrap(err, "Paths", p)
		}
	}

	return nil
}

// Validate checks that the stored information is valid.
func (g *GolangSources) Validate() error {
	if len(g.Environment) != 0 && len(g.Paths) == 0 {
		return NewFieldError("must be set if environment is set", "paths")
	}

	for _, p := range g.Paths {
		if len(p) == 0 {
			return NewFieldError("empty string is an invalid path", "paths")
		}
	}

	return nil
}
