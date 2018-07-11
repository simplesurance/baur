package baur

import (
	"path/filepath"

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
func (f *FileSrc) Resolve() ([]string, error) {
	absGlobPath := filepath.Join(f.baseDir, f.globPath)

	return fs.Glob(absGlobPath)
}
