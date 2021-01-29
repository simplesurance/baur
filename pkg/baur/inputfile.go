package baur

import (
	"path/filepath"
)

// Inputfile represent a file
type Inputfile struct {
	File
	repoRootPath string
	relPath      string
}

// NewFile returns a new file
func NewFile(repoRootPath, relPath string) *Inputfile {
	return &Inputfile{
		repoRootPath: repoRootPath,
		relPath:      relPath,
		File: File{
			AbsPath: filepath.Join(repoRootPath, relPath),
		},
	}
}

// Path returns it's absolute path
func (f *Inputfile) Path() string {
	return f.AbsPath
}

// RepoRelPath returns the path relative to the baur repository
func (f *Inputfile) RepoRelPath() string {
	return f.relPath
}

// String returns it's string representation
func (f *Inputfile) String() string {
	return f.RepoRelPath()
}
