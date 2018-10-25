package baur

import "github.com/simplesurance/baur/digest"

// BuildInput represents an input object of an application build, can be source
// files, compiler binaries etc, everything that can influence the produced
// build output
type BuildInput interface {
	Digest() (digest.Digest, error)
	String() string
	URI() string
}
