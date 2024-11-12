package cfg

// FileExcludeList specifies paths to files that are excluded as inputs.
type FileExcludeList struct {
	Paths []string `toml:"paths" comment:"Specifies files that are excluded from the Inputs.\n ExcludedFiles are processed after all other Input types.\n All Paths are relative to the application directory.\n Golang's Glob syntax (https://golang.org/pkg/path/filepath/#Match)\n and ** is supported to match files recursively."`
}

func (f *FileExcludeList) Validate() error {
	for _, path := range f.Paths {
		if len(path) == 0 {
			return newFieldError("can not be empty", "path")
		}
	}

	return nil
}

func (f *FileExcludeList) resolve(resolver Resolver) error {
	for i, p := range f.Paths {
		var err error

		if f.Paths[i], err = resolver.Resolve(p); err != nil {
			return fieldErrorWrap(err, "Paths", p)
		}
	}

	return nil
}
