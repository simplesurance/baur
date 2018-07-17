package baur

// DockerImageRef represents a DockerSource entry
type DockerImageRef struct {
	Repository string
	Digest     string
}

// Resolve resolves the DockerImageRef to a RemoteDockerImg instance
func (d *DockerImageRef) Resolve() ([]BuildInput, error) {
	return []BuildInput{NewRemoteDockerImg(d.Repository, d.Digest)}, nil
}
