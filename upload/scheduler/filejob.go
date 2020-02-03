package scheduler

import "fmt"

// FileCopyJob is an upload jobs for files to S3 repositories
type FileCopyJob struct {
	UserData interface{}
	Src      string
	Dst      string
}

// Type returns JobFileCopy
func (f *FileCopyJob) Type() JobType {
	return JobFileCopy
}

// GetUserData returns the UserData
func (f *FileCopyJob) GetUserData() interface{} {
	return f.UserData
}

// SetUserData sets the UserData
func (f *FileCopyJob) SetUserData(u interface{}) {
	f.UserData = u
}

// String returns the string representation
func (f *FileCopyJob) String() string {
	return fmt.Sprintf("%s -> %s", f.Src, f.Dst)
}
