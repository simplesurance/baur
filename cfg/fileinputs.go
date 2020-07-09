package cfg

import (
	"strings"

	"github.com/simplesurance/baur/cfg/resolver"
)

// FileInputs stores glob paths to inputs of a task.
type FileInputs struct {
	Paths []string `toml:"paths" comment:"Relative path to source files.\n Golang's Glob syntax (https://golang.org/pkg/path/filepath/#Match)\n and ** is supported to match files recursively.\n Valid variables: $ROOT, $APPNAME, $GITCOMMIT."`
}

// Merge appends the paths in other to f.
func (f *FileInputs) Merge(other *FileInputs) {
	f.Paths = append(f.Paths, other.Paths...)
}

func (f *FileInputs) Resolve(resolvers resolver.Resolver) error {
	for i, p := range f.Paths {
		var err error

		if f.Paths[i], err = resolvers.Resolve(p); err != nil {
			return FieldErrorWrap(err, "Paths", p)
		}
	}

	return nil
}

// Validate checks if the stored information is valid.
func (f *FileInputs) Validate() error {
	for _, path := range f.Paths {
		if len(path) == 0 {
			return NewFieldError("can not be empty", "path")

		}

		if strings.Count(path, "**") > 1 {
			return NewFieldError("'**' can only appear one time in a path", "path")
		}
	}

	return nil
}
