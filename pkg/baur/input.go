package baur

import "github.com/simplesurance/baur/v2/internal/digest"

// Input represents an input
type Input interface {
	Digest() (*digest.Digest, error)
	String() string
}
