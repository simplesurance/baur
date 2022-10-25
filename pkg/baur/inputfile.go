package baur

import (
	"github.com/simplesurance/baur/v3/internal/digest"
	"github.com/simplesurance/baur/v3/internal/digest/sha384"
)

// InputFile represent a file.
type InputFile struct {
	absPath     string
	repoRelPath string

	digest *digest.Digest
}

// NewInputFile creates an InputFile.
// absPath is the absolute path to the file. It is used to create the digest.
// relPath is a relative path to the file. It is used as part of the digest. To
// which base path relPath is relative is arbitrary.
func NewInputFile(absPath, relPath string) *InputFile {
	return &InputFile{
		absPath:     absPath,
		repoRelPath: relPath,
	}
}

// String returns RelPath()
func (f *InputFile) String() string {
	return f.repoRelPath
}

// RelPath returns the path of the file relative to the baur repository root.
func (f *InputFile) RelPath() string {
	return f.repoRelPath
}

// AbsPath returns the absolute path of the file.
func (f *InputFile) AbsPath() string {
	return f.absPath
}

// CalcDigest calculates the digest of the file.
// The Digest is the sha384 sum of the repoRelPath and the content of the file.
func (f *InputFile) CalcDigest() (*digest.Digest, error) {
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
func (f *InputFile) Digest() (*digest.Digest, error) {
	if f.digest != nil {
		return f.digest, nil
	}

	return f.CalcDigest()
}
