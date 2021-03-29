package cfg

// FileCopy describes a filesystem location where a task output is copied to.
type FileCopy struct {
	Path string `toml:"path" comment:"Destination directory"`
}

func (f *FileCopy) resolve(resolver Resolver) error {
	var err error

	if f.Path, err = resolver.Resolve(f.Path); err != nil {
		return fieldErrorWrap(err, "path")
	}

	return nil
}

func (f *FileCopy) validate() error {
	if f.Path == "" {
		return newFieldError("can not be empty", "path")
	}

	return nil
}
