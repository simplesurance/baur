package baur

import (
	"errors"

	"github.com/simplesurance/baur/v4/internal/digest"
	"github.com/simplesurance/baur/v4/internal/digest/sha384"
)

type InputFileOpt func(*InputFile)

// InputFile represent a file.
type InputFile struct {
	absPath                string
	repoRelPath            string
	repoRelRealPath        string
	ownerHasExecutablePerm bool

	fileHasher FileHashFn
	digest     *digest.Digest

	contentDigest *digest.Digest
}

func WithHashFn(h FileHashFn) InputFileOpt {
	return func(i *InputFile) {
		i.fileHasher = h
	}
}

func WithContentDigest(d *digest.Digest) InputFileOpt {
	return func(i *InputFile) {
		i.contentDigest = d
	}
}

// WithRealpath specifies the real path for the file.
// If one or more components of its repository relative path are symlinks (the
// file or path components), this is the repository relative path with all components resolved.
func WithRealpath(repoRelRealPath string) InputFileOpt {
	return func(i *InputFile) {
		i.repoRelRealPath = repoRelRealPath
	}
}

// NewInputFile creates an InputFile.
// absPath is the absolute path to the file. It is used to create the digest.
// relPath is a relative path to the file. It is used as part of the digest. To
// which base path relPath is relative is arbitrary.
func NewInputFile(absPath, relPath string, ownerHasExecutablePerm bool, opts ...InputFileOpt) *InputFile {
	i := InputFile{
		absPath:                absPath,
		repoRelPath:            relPath,
		ownerHasExecutablePerm: ownerHasExecutablePerm,
	}
	for _, fn := range opts {
		fn(&i)
	}
	return &i
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
// If neither a file hash functon nor the content digest was provided on
// construction of f, errors.ErrUnsupported is returned.
func (f *InputFile) CalcDigest() (*digest.Digest, error) {
	if f.contentDigest == nil && f.fileHasher == nil {
		return nil, errors.ErrUnsupported
	}

	h := sha384.New()

	if err := h.AddBytes([]byte("P:")); err != nil {
		return nil, err
	}
	if err := h.AddBytes([]byte(f.repoRelPath)); err != nil {
		return nil, err
	}

	if err := h.AddBytes([]byte("R:")); err != nil {
		return nil, err
	}

	if f.repoRelRealPath != "" {
		if err := h.AddBytes([]byte(f.repoRelRealPath)); err != nil {
			return nil, err
		}
	}

	if f.contentDigest == nil {
		d, err := f.fileHasher(f.absPath)
		if err != nil {
			return nil, err
		}
		f.contentDigest = d
	}

	if f.ownerHasExecutablePerm {
		if err := h.AddBytes([]byte("E:1")); err != nil {
			return nil, err
		}
	} else {
		if err := h.AddBytes([]byte("E:0")); err != nil {
			return nil, err
		}
	}

	if err := h.AddBytes([]byte("C:")); err != nil {
		return nil, err
	}
	if err := h.AddBytes(f.contentDigest.Sum); err != nil {
		return nil, err
	}

	return h.Digest(), nil
}

// Digest returns the previous calculated digest.
// If the digest wasn't calculated yet, CalcDigest() is called.
func (f *InputFile) Digest() (*digest.Digest, error) {
	if f.digest != nil {
		return f.digest, nil
	}

	return f.CalcDigest()
}
