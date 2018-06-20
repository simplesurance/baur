package baur

import "github.com/simplesurance/baur/docker"

// ArtifactBackends contains a list of backends that are required to interact
// with artifacts
type ArtifactBackends struct {
	DockerClt *docker.Client
}
