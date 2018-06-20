package baur

import (
	"github.com/simplesurance/baur/digest"
	"github.com/simplesurance/baur/digest/file"
	"github.com/simplesurance/baur/fs"
	"github.com/simplesurance/baur/upload"
)

// FileArtifact is a file build artifact
type FileArtifact struct {
	RelPath   string
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
	return f.Name()
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
	return file.SHA256Digest(f.LocalPath())
}

// Size returns the size of the file in bytes
func (f *FileArtifact) Size(_ *ArtifactBackends) (int64, error) {
	return fs.FileSize(f.LocalPath())
}
