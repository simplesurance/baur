package upload

// JobType describes the type of a job
type JobType int

const (
	_ JobType = iota
	// JobS3 is the type for S3 file upload jobs
	JobS3
	// JobDocker is the type for Docker container uploader jobs
	JobDocker
)

// Job is the interface for upload jobs
type Job interface {
	LocalPath() string
	RemoteDest() string
	Type() JobType
	GetUserData() interface{}
	SetUserData(interface{})
	String() string
}
