package cfg

import (
	"strings"

	"github.com/simplesurance/baur/v1/pkg/cfg/resolver"
)

// FileInputs stores glob paths to inputs of a task.
type FileInputs struct {
	Paths          []string `toml:"paths" comment:"Relative path to source files.\n Golang's Glob syntax (https://golang.org/pkg/path/filepath/#Match)\n and ** is supported to match files recursively."`
	Optional       bool     `toml:"optional" comment:"If true, baur will not fail if a Path resolves to 0 files."`
	GitTrackedOnly bool     `toml:"git_tracked_only" comment:"Ignore files that are not tracked in git."`
}

func (f *FileInputs) resolve(resolvers resolver.Resolver) error {
	for i, p := range f.Paths {
		var err error

		if f.Paths[i], err = resolvers.Resolve(p); err != nil {
			return fieldErrorWrap(err, "Paths", p)
		}
	}

	return nil
}

// validate checks if the stored information is valid.
func (f *FileInputs) validate() error {
	for _, path := range f.Paths {
		if len(path) == 0 {
			return newFieldError("can not be empty", "path")

		}

		if strings.Count(path, "**") > 1 {
			return newFieldError("'**' can only appear one time in a path", "path")
		}
	}

	return nil
}
