package baur

import (
	"github.com/simplesurance/baur/fs"
	"github.com/simplesurance/baur/upload"
)

// FileArtifact is a file build artifact
type FileArtifact struct {
	Path      string
	DestFile  string
	UploadURL string
}

// Exists returns true if the artifact exist
func (f *FileArtifact) Exists() bool {
	return fs.FileExists(f.Path)
}

// String returns the String representation
func (f *FileArtifact) String() string {
	return f.Path
}

// UploadJob returns a upload.DockerJob for the artifact
func (f *FileArtifact) UploadJob() (upload.Job, error) {
	return &upload.S3Job{
		DestURL:  f.UploadURL,
		FilePath: f.Path,
	}, nil
}

// LocalPath returns the local path to the artifact
func (f *FileArtifact) LocalPath() string {
	return f.Path
}

// UploadDestination returns the upload destination
func (f *FileArtifact) UploadDestination() string {
	return f.UploadURL
}
