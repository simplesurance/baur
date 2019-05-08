package baur

import (
	"github.com/simplesurance/baur/digest"
	"github.com/simplesurance/baur/digest/sha384"
	"github.com/simplesurance/baur/fs"
	"github.com/simplesurance/baur/upload/scheduler"
)

// FileArtifact is a file build artifact
type FileArtifact struct {
	RelPath   string
	Path      string
	DestFile  string
	UploadURL string
	uploadJob scheduler.Job
}

// Exists returns true if the artifact exist
func (f *FileArtifact) Exists() bool {
	return fs.FileExists(f.Path)
}

// String returns the String representation
func (f *FileArtifact) String() string {
	return f.RelPath
}

// UploadJob returns a upload.DockerJob for the artifact
func (f *FileArtifact) UploadJob() (scheduler.Job, error) {
	return f.uploadJob, nil
}

// LocalPath returns the local path to the artifact
func (f *FileArtifact) LocalPath() string {
	return f.Path
}

// Name returns the path to the artifact relatively to application dir
func (f *FileArtifact) Name() string {
	return f.RelPath
}

// UploadDestination returns the upload destination
func (f *FileArtifact) UploadDestination() string {
	return f.UploadURL
}

// Digest returns the file digest
func (f *FileArtifact) Digest() (*digest.Digest, error) {
	sha := sha384.New()

	err := sha.AddFile(f.LocalPath())
	if err != nil {
		return nil, err
	}

	return sha.Digest(), err
}

// Size returns the size of the file in bytes
func (f *FileArtifact) Size(_ *BuildOutputBackends) (int64, error) {
	return fs.FileSize(f.LocalPath())
}

// Type returns "File"
func (f *FileArtifact) Type() string {
	return "File"
}
