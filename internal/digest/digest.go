package digest

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

// Algorithm describes the digest algorithm
type Algorithm int

const (
	_ Algorithm = iota
	// SHA256 is the sha256 algorithm
	SHA256
	// SHA384 is the sha384 algorithm
	SHA384
)

// String returns the textual representation
func (t Algorithm) String() string {
	switch t {
	case SHA256:
		return "sha256"

	case SHA384:
		return "sha384"
	default:
		return "undefined"
	}
}

// Digest contains a checksum
type Digest struct {
	Sum       []byte
	Algorithm Algorithm
}

// String returns '<Algorithm>:<hash>'
func (d *Digest) String() string {
	return fmt.Sprintf("%s:%s", d.Algorithm, hex.EncodeToString(d.Sum))
}

// FromString converts a "<Algorithm>:<hash> string to Digest
func FromString(in string) (*Digest, error) {
	var algorithm Algorithm

	spl := strings.Split(strings.TrimSpace(in), ":")
	if len(spl) != 2 {
		return nil, errors.New("invalid format, must contain exactly 1 ':'")
	}

	switch a := strings.ToLower(spl[0]); a {
	case "sha256":
		if len(spl[1]) != 64 {
			return nil, fmt.Errorf("hash length is %d, expected length 64", len(spl[1]))
		}

		algorithm = SHA256
	case "sha384":
		if len(spl[1]) != 96 {
			return nil, fmt.Errorf("hash length is %d, expected length 96", len(spl[1]))
		}

		algorithm = SHA384
	default:
		return nil, fmt.Errorf("unsupported format %q", a)
	}

	sum, err := hex.DecodeString(spl[1])
	if err != nil {
		return nil, fmt.Errorf("converting string sum to hex failed: %w", err)
	}

	return &Digest{
		Sum:       sum,
		Algorithm: algorithm,
	}, nil
}
