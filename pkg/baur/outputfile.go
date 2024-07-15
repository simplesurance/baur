package baur

import (
	"github.com/simplesurance/baur/v5/internal/digest"
	"github.com/simplesurance/baur/v5/internal/digest/sha384"
	"github.com/simplesurance/baur/v5/internal/fs"
)

// OutputFile is a file created by a task run.
type OutputFile struct {
	name            string
	absPath         string
	UploadsS3       []*UploadInfoS3
	UploadsFilecopy []*UploadInfoFileCopy

	digest *digest.Digest
}

func NewOutputFile(name, absPath string, s3uploads []*UploadInfoS3, filecopyUploads []*UploadInfoFileCopy) *OutputFile {
	return &OutputFile{
		name:            name,
		absPath:         absPath,
		UploadsS3:       s3uploads,
		UploadsFilecopy: filecopyUploads,
	}
}

func (f *OutputFile) String() string {
	return "file: " + f.name
}

func (f *OutputFile) AbsPath() string {
	return f.absPath
}

func (f *OutputFile) Name() string {
	return f.name
}

func (f *OutputFile) Type() OutputType {
	return FileOutput
}

// CalcDigest calculates the digest of the file.
// The Digest is the sha384 sum of the content of the file.
func (f *OutputFile) CalcDigest() (*digest.Digest, error) {
	sha := sha384.New()

	err := sha.AddFile(f.absPath)
	if err != nil {
		return nil, err
	}

	f.digest = sha.Digest()

	return f.digest, nil
}

// Digest returns the previous calculated digest.
// If the digest wasn't calculated yet, CalcDigest() is called.
func (f *OutputFile) Digest() (*digest.Digest, error) {
	if f.digest != nil {
		return f.digest, nil
	}

	return f.CalcDigest()
}

func (f *OutputFile) Exists() (bool, error) {
	return fs.FileExists(f.absPath), nil
}

func (f *OutputFile) SizeBytes() (uint64, error) {
	size, err := fs.FileSize(f.absPath)
	if err != nil {
		return 0, err
	}

	return uint64(size), nil
}
