package baur

import "fmt"

// DockerImageRef represents a DockerSource entry
type DockerImageRef struct {
	Repository string
	Digest     string
}

// Resolve resolves the DockerImageRef to a RemoteDockerImg instance
func (d *DockerImageRef) Resolve() ([]BuildInput, error) {
	return []BuildInput{NewRemoteDockerImg(d.Repository, d.Digest)}, nil
}

// Type returns the type of resolver
func (d *DockerImageRef) Type() string {
	return "DockerImage"
}

// Path returns the path that is resolved
func (d *DockerImageRef) String() string {
	return fmt.Sprintf("%s:%s", d.Repository, d.Digest)
}
