package digest

import (
	"errors"
	"fmt"
	"strings"
)

// Algorithm describes the digest algorithm
type Algorithm int

const (
	_ Algorithm = iota
	// SHA256 is a sha256 checksum
	SHA256
)

// String returns the textual representation
func (t Algorithm) String() string {
	switch t {
	case SHA256:
		return "sha256"
	default:
		return "undefined"
	}
}

// Digest contains a checksum
type Digest struct {
	Sum       string
	Algorithm Algorithm
}

// String returns '<Algorithm>:<checksum>'
func (d *Digest) String() string {
	return fmt.Sprintf("%s:%s", d.Algorithm, d.Sum)
}

// FromString converts a "sha256:<hash> string to Digest
func FromString(in string) (*Digest, error) {
	spl := strings.Split(strings.TrimSpace(in), ":")
	if len(spl) != 2 {
		return nil, errors.New("invalid format, must contain exactly 1 ':'")
	}

	if spl[0] != "sha256" {
		return nil, errors.New("unsupported format %q")
	}

	if len(spl[1]) != 64 {
		return nil, fmt.Errorf("hash length is %d, expected length 64", len(spl[1]))
	}

	return &Digest{
		Sum:       spl[1],
		Algorithm: SHA256,
	}, nil
}
