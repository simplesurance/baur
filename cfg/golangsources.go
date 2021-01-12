package cfg

import (
	"github.com/simplesurance/baur/v1/cfg/resolver"
)

// GolangSources specifies inputs for Golang Applications
type GolangSources struct {
	Queries     []string `toml:"queries" comment:"Queries specify the source files or packages of which the dependencies are resolved.\n Format:\n \tfile=<RELATIVE-PATH>\n \tfileglob=<GLOB-PATTERN>\t -> Supports double-star\n \tEverything else is passed directly to underlying build tool (go list by default).\n \tSee also the patterns described at:\n \t<https://github.com/golang/tools/blob/bc8aaaa29e0665201b38fa5cb5d47826788fa249/go/packages/doc.go#L17>.\n Files from Golang's stdlib are ignored."`
	Environment []string `toml:"environment" comment:"Environment to use when discovering Golang source files.\n Variables from the current environment are not used.\n These shoul be environment variables understood by the Golang tools, like GOPATH, GOFLAGS, etc.\n If empty the default Go environment is used"`
	BuildFlags  []string `toml:"build_flags" comment:"List of command-line flags to be passed through to the build system's query tool."`
	Tests       bool     `toml:"tests" comment:"If true queries are resolved to test files, otherwise testfiles are ignored."`
}

func (g *GolangSources) resolve(resolvers resolver.Resolver) error {
	for i, env := range g.Environment {
		var err error

		if g.Environment[i], err = resolvers.Resolve(env); err != nil {
			return fieldErrorWrap(err, "Environment", env)
		}
	}

	for i, q := range g.Queries {
		var err error

		if g.Queries[i], err = resolvers.Resolve(q); err != nil {
			return fieldErrorWrap(err, "Paths", q)
		}
	}

	for i, f := range g.BuildFlags {
		var err error

		if g.BuildFlags[i], err = resolvers.Resolve(f); err != nil {
			return fieldErrorWrap(err, "build_flags", f)
		}
	}

	return nil
}

// validate checks that the stored information is valid.
func (g *GolangSources) validate() error {
	if (len(g.Environment) != 0 || len(g.BuildFlags) != 0 || g.Tests) &&
		len(g.Queries) == 0 {
		return newFieldError("must be set if environment, build_flags or tests is set", "query")
	}

	for _, q := range g.Queries {
		if len(q) == 0 {
			return newFieldError("empty string is an invalid query", "query")
		}
	}

	return nil
}
