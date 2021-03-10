package baur

import (
	"path/filepath"

	"github.com/simplesurance/baur/v1/internal/digest"
	"github.com/simplesurance/baur/v1/internal/digest/sha384"
)

// Inputfile represent a file
type Inputfile struct {
	absPath     string
	repoRelPath string

	digest *digest.Digest
}

// NewInputFile returns a new input file
func NewInputFile(repoRootPath, relPath string) *Inputfile {
	return &Inputfile{
		absPath:     filepath.Join(repoRootPath, relPath),
		repoRelPath: relPath,
	}
}

// String returns it's string representation
func (f *Inputfile) String() string {
	return f.repoRelPath
}

func (f *Inputfile) AbsPath() string {
	return f.absPath
}

// CalcDigest calculates the digest of the file.
// The Digest is the sha384 sum of the repoRelPath and the content of the file.
func (f *Inputfile) CalcDigest() (*digest.Digest, error) {
	sha := sha384.New()

	err := sha.AddBytes([]byte(f.repoRelPath))
	if err != nil {
		return nil, err
	}

	err = sha.AddFile(f.absPath)
	if err != nil {
		return nil, err
	}

	f.digest = sha.Digest()

	return f.digest, nil
}

// Digest returns the previous calculated digest.
// If the digest wasn't calculated yet, CalcDigest() is called.
func (f *Inputfile) Digest() (*digest.Digest, error) {
	if f.digest != nil {
		return f.digest, nil
	}

	return f.CalcDigest()
}
