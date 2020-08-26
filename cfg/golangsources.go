package cfg

import (
	"github.com/simplesurance/baur/v1/cfg/resolver"
)

// GolangSources specifies inputs for Golang Applications
type GolangSources struct {
	Environment []string `toml:"environment" comment:"Environment to use when discovering Golang source files\n This are environment variables understood by the Golang tools, like GOPATH, GOFLAGS, etc.\n If empty the default Go environment is used.\n Valid variables: $ROOT, $APPNAME"`
	Queries     []string `toml:"queries" comment:"Specifies the source files or packages of which the dependencies are resolved.\n Queries are passed to the underlying build tool, go list normally.\n Therefore it supports the regulard golang packages pattern (see go help packages).\n When another build tool is used the query syntax described at <https://github.com/golang/tools/blob/bc8aaaa29e0665201b38fa5cb5d47826788fa249/go/packages/doc.go#L17> must be used.\n. Files from Golang's stdlib are ignored.\n Valid variables: $ROOT, $APPNAME."`
}

// Merge merges the two GolangSources structs
func (g *GolangSources) Merge(other *GolangSources) {
	g.Queries = append(g.Queries, other.Queries...)
	g.Environment = append(g.Environment, other.Environment...)
}

func (g *GolangSources) Resolve(resolvers resolver.Resolver) error {
	for i, env := range g.Environment {
		var err error

		if g.Environment[i], err = resolvers.Resolve(env); err != nil {
			return FieldErrorWrap(err, "Environment", env)
		}
	}

	for i, q := range g.Queries {
		var err error

		if g.Queries[i], err = resolvers.Resolve(q); err != nil {
			return FieldErrorWrap(err, "Paths", q)
		}
	}

	return nil
}

// Validate checks that the stored information is valid.
func (g *GolangSources) Validate() error {
	if len(g.Environment) != 0 && len(g.Queries) == 0 {
		return NewFieldError("must be set if environment is set", "query")
	}

	for _, q := range g.Queries {
		if len(q) == 0 {
			return NewFieldError("empty string is an invalid query", "query")
		}
	}

	return nil
}
