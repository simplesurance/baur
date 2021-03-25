package baur

import "github.com/simplesurance/baur/v2/internal/digest"

// Input represents an input
type Input interface {
	Digest() (*digest.Digest, error)
	String() string
}

// InputAddStrIfNotEmpty returns a new slice that contains in and str as an
// InputString object, if str is not an empty string.
// If it is, in is returned.
func InputAddStrIfNotEmpty(in []Input, str string) []Input {
	if str == "" {
		return in
	}

	return append(in, NewInputString(str))
}
