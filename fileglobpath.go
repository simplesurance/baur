package baur

import (
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/simplesurance/baur/fs"
)

//FileGlobPath is Source file of an application represented by a glob path
type FileGlobPath struct {
	repositoryRootPath string
	relAppPath         string
	globPath           string
}

// NewFileGlobPath returns a new FileGlobPath object
func NewFileGlobPath(repositoryRootPath, relAppPath, glob string) *FileGlobPath {
	return &FileGlobPath{
		repositoryRootPath: repositoryRootPath,
		relAppPath:         relAppPath,
		globPath:           glob,
	}
}

// Resolve returns a list of files that are matching the glob path of the
// FileGlobPath
func (f *FileGlobPath) Resolve() ([]BuildInput, error) {
	absGlobPath := filepath.Join(f.repositoryRootPath, f.relAppPath, f.globPath)

	paths, err := fs.Glob(absGlobPath)
	if err != nil {
		return nil, err
	}

	if len(paths) == 0 {
		return nil, errors.New("glob matched 0 files")
	}

	res := make([]BuildInput, 0, len(paths))
	for _, p := range paths {
		relPath, err := filepath.Rel(f.repositoryRootPath, p)
		if err != nil {
			return nil, errors.Wrapf(err, "converting %q to relpath with basedir %q failed", p, f.repositoryRootPath)
		}

		res = append(res, NewFile(f.repositoryRootPath, relPath))
	}

	return res, nil
}
