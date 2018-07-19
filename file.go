package baur

import (
	"fmt"
	"path/filepath"

	"github.com/simplesurance/baur/digest"
	"github.com/simplesurance/baur/digest/sha384"
)

// File represent a file
type File struct {
	repoRootPath string
	relPath      string
	absPath      string
	digest       *digest.Digest
}

// NewFile returns a new file
func NewFile(repoRootPath, relPath string) *File {
	return &File{
		repoRootPath: repoRootPath,
		relPath:      relPath,
		absPath:      filepath.Join(repoRootPath, relPath),
	}
}

// Digest returns a digest of the file
func (f *File) Digest() (digest.Digest, error) {
	if f.digest != nil {
		return *f.digest, nil
	}

	d, err := sha384.File(filepath.Join(f.absPath))
	if err != nil {
		return digest.Digest{}, nil
	}

	f.digest = d

	return *f.digest, nil
}

// Path returns it's absolute path
func (f *File) Path() string {
	return f.absPath
}

// RepoRelPath returns the path relative to the baur repository
func (f *File) RepoRelPath() string {
	return f.relPath
}

// URL returns an URL
func (f *File) URL() string {
	return fmt.Sprintf("file://%s", f.relPath)
}

// String returns it's string representation
func (f *File) String() string {
	return f.Path()
}
