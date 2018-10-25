package upload

import "fmt"

// DockerJob is a docker container upload job
type DockerJob struct {
	UserData   interface{}
	ImageID    string
	Repository string
	Tag        string
}

// LocalPath returns the image id of the container
func (d *DockerJob) LocalPath() string {
	return d.ImageID
}

// RemoteDest returns the upload path in the docker registry
func (d *DockerJob) RemoteDest() string {
	return d.Repository + ":" + d.Tag
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
	return fmt.Sprintf("docker image: %s:%s", d.Repository, d.Tag)
}
