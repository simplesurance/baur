package cfg

import (
	"strings"
)

// FileInputs stores glob paths to inputs of a task.
type FileInputs struct {
	Paths          []string `toml:"paths" comment:"Glob patterns that match files.\n All Paths are relative to the application directory.\n Golang's Glob syntax (https://golang.org/pkg/path/filepath/#Match)\n and ** is supported to match files recursively."`
	Optional       bool     `toml:"optional" comment:"When optional is true a path pattern that matches 0 files will not cause an error."`
	GitTrackedOnly bool     `toml:"git_tracked_only" comment:"Only resolve to files that are part of the Git repository."`
}

func (f *FileInputs) resolve(resolver Resolver) error {
	for i, p := range f.Paths {
		var err error

		if f.Paths[i], err = resolver.Resolve(p); err != nil {
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
