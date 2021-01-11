package baur

import (
	"github.com/simplesurance/baur/v1/internal/digest"
	"github.com/simplesurance/baur/v1/internal/fs"
)

// OutputFile is a file created by a task run.
type OutputFile struct {
	*File
	name            string
	UploadsS3       []*UploadInfoS3
	UploadsFilecopy []*UploadInfoFileCopy
}

func NewOutputFile(name, absPath string, s3uploads []*UploadInfoS3, filecopyUploads []*UploadInfoFileCopy) *OutputFile {
	return &OutputFile{
		name:            name,
		File:            &File{AbsPath: absPath},
		UploadsS3:       s3uploads,
		UploadsFilecopy: filecopyUploads,
	}
}

func (f *OutputFile) String() string {
	return "file: " + f.name
}

func (f *OutputFile) Name() string {
	return f.name
}

func (f *OutputFile) Type() OutputType {
	return FileOutput
}

func (f *OutputFile) Digest() (*digest.Digest, error) {
	return f.File.Digest()
}

func (f *OutputFile) Exists() (bool, error) {
	return fs.FileExists(f.AbsPath), nil
}

func (f *OutputFile) Size() (uint64, error) {
	size, err := fs.FileSize(f.AbsPath)
	if err != nil {
		return 0, err
	}

	return uint64(size), nil
}
