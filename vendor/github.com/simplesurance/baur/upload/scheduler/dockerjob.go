package scheduler

import "fmt"

// DockerJob is a docker container upload job
type DockerJob struct {
	UserData   interface{}
	ImageID    string
	Registry   string
	Repository string
	Tag        string
}

// Type returns the JobDocker
func (d *DockerJob) Type() JobType {
	return JobDocker
}

// GetUserData returns the UserData
func (d *DockerJob) GetUserData() interface{} {
	return d.UserData
}

// SetUserData sets the UserData
func (d *DockerJob) SetUserData(u interface{}) {
	d.UserData = u
}

// String returns the string representation
func (d *DockerJob) String() string {
	if d.Registry == "" {
		return fmt.Sprintf("docker: %s:%s", d.Repository, d.Tag)
	}

	return fmt.Sprintf("docker: %s/%s:%s", d.Registry, d.Repository, d.Tag)
}
