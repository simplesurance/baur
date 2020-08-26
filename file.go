package baur

import (
	"github.com/simplesurance/baur/v1/internal/digest"
	"github.com/simplesurance/baur/v1/internal/digest/sha384"
)

// File represent a file
type File struct {
	AbsPath string
	digest  *digest.Digest
}

// CalcDigest calculates the digest of the file, saves it and returns it.
func (f *File) CalcDigest() (*digest.Digest, error) {
	sha := sha384.New()

	err := sha.AddBytes([]byte(f.AbsPath))
	if err != nil {
		return nil, err
	}

	err = sha.AddFile(f.AbsPath)
	if err != nil {
		return nil, err
	}

	f.digest = sha.Digest()

	return f.digest, nil
}

// Digest returns the previous calculated digest.
// If the digest wasn't calculated yet, CalcDigest() is called and it's return
// values are returned.
func (f *File) Digest() (*digest.Digest, error) {
	if f.digest != nil {
		return f.digest, nil
	}

	return f.CalcDigest()
}

func (f *File) Path() string {
	return f.AbsPath
}
