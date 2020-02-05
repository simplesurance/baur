package cfg

import "github.com/simplesurance/baur/cfg/resolver"

// FileCopy describes a filesystem location where a task output is copied to.
type FileCopy struct {
	Path string `toml:"path" comment:"Destination directory\n Valid variables: $ROOT, $APPNAME, $GITCOMMIT, $UUID." commented:"true"`
}

// IsEmpty returns true if FileCopy is empty
func (f *FileCopy) IsEmpty() bool {
	return len(f.Path) == 0
}

func (f *FileCopy) Resolve(resolvers resolver.Resolver) error {
	var err error

	if f.Path, err = resolvers.Resolve(f.Path); err != nil {
		return FieldErrorWrap(err, "path")
	}

	return nil
}
