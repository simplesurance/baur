package baur

// DockerSrc represents a DockerSource entry
type DockerSrc struct {
	Repository string
	Digest     string
}

// Resolve resolves the DockerSrc to a RemoteDockerImg instance
func (d *DockerSrc) Resolve() ([]BuildInput, error) {
	return []BuildInput{NewRemoteDockerImg(d.Repository, d.Digest)}, nil
}
