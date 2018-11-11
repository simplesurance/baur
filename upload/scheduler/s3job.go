package scheduler

import "fmt"

// S3Job is an upload jobs for files to S3 repositories
type S3Job struct {
	UserData interface{}
	FilePath string
	DestURL  string
}

// LocalPath returns the local path of the file that is uploaded
func (s *S3Job) LocalPath() string {
	return s.FilePath
}

// RemoteDest returns the path in S3
func (s *S3Job) RemoteDest() string {
	return s.DestURL
}

// Type returns JobS3
func (s *S3Job) Type() JobType {
	return JobS3
}

// GetUserData returns the UserData
func (s *S3Job) GetUserData() interface{} {
	return s.UserData
}

// SetUserData sets the UserData
func (s *S3Job) SetUserData(u interface{}) {
	s.UserData = u
}

// String returns the string representation
func (s *S3Job) String() string {
	return fmt.Sprintf("%s -> %s", s.FilePath, s.DestURL)
}
