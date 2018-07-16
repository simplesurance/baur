package baur

import (
	"path/filepath"

	"github.com/simplesurance/baur/digest"
	"github.com/simplesurance/baur/digest/sha256"
)

// File represent a file
type File struct {
	rootDir string
	relPath string
	absPath string
}

// NewFile returns a new file
func NewFile(rootDir, relPath string) *File {
	return &File{
		rootDir: rootDir,
		relPath: relPath,
		absPath: filepath.Join(rootDir, relPath),
	}
}

// Digest returns a digest of the file
func (f *File) Digest() (*digest.Digest, error) {
	return sha256.File(filepath.Join(f.absPath))
}

// Path returns it's absolute path
func (f *File) Path() string {
	return f.absPath
}

// RelPath returns the relative path
func (f *File) RelPath() string {
	return f.relPath
}
