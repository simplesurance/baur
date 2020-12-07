package cfg

import "github.com/simplesurance/baur/v1/cfg/resolver"

// GitFileInputs describes source files that are in the git repository by git
// pathnames
type GitFileInputs struct {
	Paths    []string `toml:"paths" comment:"Relative paths to source files.\n Only files tracked by Git that are not in the .gitignore file are matched.\n The same patterns that git ls-files supports can be used.\n Valid variables: $ROOT, $APPNAME."`
	Optional bool     `toml:"optional" comment:"If true, baur will not fail if a Path does not resolve to a file."`
}

func (g *GitFileInputs) resolve(resolvers resolver.Resolver) error {
	for i, p := range g.Paths {
		var err error

		if g.Paths[i], err = resolvers.Resolve(p); err != nil {
			return fieldErrorWrap(err, "Paths", p)
		}
	}

	return nil
}
