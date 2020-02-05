package cfg

import (
	"github.com/simplesurance/baur/cfg/resolver"
)

// S3Upload contains S3 upload information
type S3Upload struct {
	Bucket   string `toml:"bucket" comment:"Bucket name, valid variables: $APPNAME, $UUID, $GITCOMMIT" commented:"true"`
	DestFile string `toml:"dest_file" comment:"Remote File Name, valid variables: $ROOT, $APPNAME, $UUID, $GITCOMMIT" commented:"true"`
}

// IsEmpty returns true if S3Upload is empty
func (s *S3Upload) IsEmpty() bool {
	return len(s.Bucket) == 0 && len(s.DestFile) == 0
}

func (s *S3Upload) Resolve(resolvers resolver.Resolver) error {
	var err error

	if s.Bucket, err = resolvers.Resolve(s.Bucket); err != nil {
		return FieldErrorWrap(err, "bucket")
	}

	if s.DestFile, err = resolvers.Resolve(s.DestFile); err != nil {
		return FieldErrorWrap(err, "dest_file")
	}

	return nil
}

// Validate validates a [[Task.Output.File]] section
func (s *S3Upload) Validate() error {
	if s.IsEmpty() {
		return nil
	}

	if len(s.DestFile) == 0 {
		return NewFieldError("can not be empty", "destfile")
	}

	if len(s.Bucket) == 0 {
		return NewFieldError("can not be empty", "bucket")
	}

	return nil
}
