package scheduler

import "fmt"

// S3Job is an upload jobs for files to S3 repositories
type S3Job struct {
	UserData interface{}
	FilePath string
	DestURL  string
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
