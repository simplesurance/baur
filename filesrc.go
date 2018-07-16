package baur

import (
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/simplesurance/baur/fs"
)

//FileSrc is Source file of an application represented by a glob path
type FileSrc struct {
	baseDir  string
	globPath string
}

// NewFileSrc returns a new FileSrc object
func NewFileSrc(baseDir, glob string) *FileSrc {
	return &FileSrc{
		baseDir:  baseDir,
		globPath: glob,
	}
}

// Resolve returns a list of files that are matching the glob path of the
// FileSrc
func (f *FileSrc) Resolve() ([]*File, error) {
	absGlobPath := filepath.Join(f.baseDir, f.globPath)

	paths, err := fs.Glob(absGlobPath)
	if err != nil {
		return nil, err
	}

	if len(paths) == 0 {
		return nil, errors.New("glob matched 0 files")
	}

	res := make([]*File, 0, len(paths))
	for _, p := range paths {
		relPath, err := filepath.Rel(f.baseDir, p)
		if err != nil {
			return nil, errors.Wrapf(err, "converting %q to relpath with basedir %q failed", p, f.baseDir)
		}

		res = append(res, NewFile(f.baseDir, relPath))
	}

	return res, nil
}
