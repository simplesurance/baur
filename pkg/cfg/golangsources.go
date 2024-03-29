package cfg

// GolangSources specifies inputs for Golang Applications
type GolangSources struct {
	// if attributes are added/removed or modified, the input resolver
	// cache *must* be adapted to ensure that the caching logic respects
	// the attribute change.
	Queries     []string `toml:"queries" comment:"Go package queries, the source files of matching packages and\n their imported packages are resolved to files.\n Format:\n \tfile=<RELATIVE-PATH>\n \tfileglob=<GLOB-PATTERN>\t -> Supports double-star\n \tEverything else is passed to the Go query tool (go list by default).\n \tSee also the patterns described at:\n \t<https://github.com/golang/tools/blob/bc8aaaa29e0665201b38fa5cb5d47826788fa249/go/packages/doc.go#L17>.\n Files from Golang's stdlib are ignored."`
	Environment []string `toml:"environment" comment:"Environment when running the go query tool."`
	BuildFlags  []string `toml:"build_flags" comment:"List of command-line flags to be passed through to the Go query tool."`
	Tests       bool     `toml:"tests" comment:"If true queries are resolved to test files, otherwise testfiles are ignored."`
}

func (g *GolangSources) resolve(resolver Resolver) error {
	for i, env := range g.Environment {
		var err error

		if g.Environment[i], err = resolver.Resolve(env); err != nil {
			return fieldErrorWrap(err, "Environment", env)
		}
	}

	for i, q := range g.Queries {
		var err error

		if g.Queries[i], err = resolver.Resolve(q); err != nil {
			return fieldErrorWrap(err, "Paths", q)
		}
	}

	for i, f := range g.BuildFlags {
		var err error

		if g.BuildFlags[i], err = resolver.Resolve(f); err != nil {
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
