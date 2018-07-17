package baur

import "github.com/simplesurance/baur/docker"

// BuildOutputBackends contains a list of backends that are required to interact
// with artifacts
type BuildOutputBackends struct {
	DockerClt *docker.Client
}
