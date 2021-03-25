package cfg

import (
	"github.com/simplesurance/baur/v2/pkg/cfg/resolver"
)

// S3Upload contains S3 upload information
type S3Upload struct {
	Bucket string `toml:"bucket"`
	Key    string `toml:"key"`
}

func (s *S3Upload) resolve(resolvers resolver.Resolver) error {
	var err error

	if s.Bucket, err = resolvers.Resolve(s.Bucket); err != nil {
		return fieldErrorWrap(err, "bucket")
	}

	if s.Key, err = resolvers.Resolve(s.Key); err != nil {
		return fieldErrorWrap(err, "key")
	}

	return nil
}

// validate validates a [[Task.Output.File]] section
func (s *S3Upload) validate() error {
	if len(s.Key) == 0 {
		return newFieldError("can not be empty", "key")
	}

	if len(s.Bucket) == 0 {
		return newFieldError("can not be empty", "bucket")
	}

	return nil
}
