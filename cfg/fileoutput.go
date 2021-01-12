package cfg

import (
	"github.com/simplesurance/baur/v1/cfg/resolver"
)

// FileOutput describes where a file output is stored.
type FileOutput struct {
	Path     string     `toml:"path" comment:"Path relative to the application directory."`
	FileCopy []FileCopy `comment:"Copy the file to a local directory."`
	S3Upload []S3Upload `comment:"Upload the file to S3."`
}

func (f *FileOutput) resolve(resolvers resolver.Resolver) error {
	var err error

	if f.Path, err = resolvers.Resolve(f.Path); err != nil {
		return fieldErrorWrap(err, "path")
	}

	for i, fc := range f.FileCopy {
		if err = fc.resolve(resolvers); err != nil {
			return fieldErrorWrap(err, "FileCopy")
		}

		f.FileCopy[i] = fc
	}

	for i, s3 := range f.S3Upload {
		if err = s3.resolve(resolvers); err != nil {
			return fieldErrorWrap(err, "S3Upload")
		}

		f.S3Upload[i] = s3
	}

	return nil
}

// validate checks that the stored information is valid.
func (f *FileOutput) validate() error {
	if len(f.Path) == 0 {
		return newFieldError("can not be empty", "path")
	}

	for _, s3 := range f.S3Upload {
		err := s3.validate()
		if err != nil {
			return fieldErrorWrap(err, "S3Upload")
		}
	}

	return nil
}
