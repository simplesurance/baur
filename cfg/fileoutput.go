package cfg

import (
	"github.com/simplesurance/baur/cfg/resolver"
)

// FileOutput describes where a file output is stored.
type FileOutput struct {
	Path     string   `toml:"path" comment:"Path relative to the application directory.\n Valid variables: $ROOT, $APPNAME, $GITCOMMIT."`
	FileCopy FileCopy `comment:"Copy the file to a local directory."`
	S3Upload S3Upload `comment:"Upload the file to S3."`
}

func (f *FileOutput) Resolve(resolvers resolver.Resolver) error {
	var err error

	if f.Path, err = resolvers.Resolve(f.Path); err != nil {
		return FieldErrorWrap(err, "path")
	}

	if err = f.FileCopy.Resolve(resolvers); err != nil {
		return FieldErrorWrap(err, "FileCopy")
	}

	if err = f.S3Upload.Resolve(resolvers); err != nil {
		return FieldErrorWrap(err, "S3Upload")
	}

	return nil
}

// Validate checks that the stored information is valid.
func (f *FileOutput) Validate() error {
	if len(f.Path) == 0 {
		return NewFieldError("can not be empty", "path")
	}

	return f.S3Upload.Validate()
}

// IsEmpty returns true if the object stores no data.
func (f *FileOutput) IsEmpty() bool {
	return f.Path == "" && f.S3Upload.IsEmpty() && f.FileCopy.IsEmpty()
}
