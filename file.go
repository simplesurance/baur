package baur

import (
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

	sha := sha384.New()

	err := sha.AddBytes([]byte(f.relPath))
	if err != nil {
		return digest.Digest{}, err
	}

	err = sha.AddFile(filepath.Join(f.absPath))
	if err != nil {
		return digest.Digest{}, err
	}

	f.digest = sha.Digest()

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

// URI calls RepoRelPath()
func (f *File) URI() string {
	return f.RepoRelPath()
}

// String returns it's string representation
func (f *File) String() string {
	return f.RepoRelPath()
}
