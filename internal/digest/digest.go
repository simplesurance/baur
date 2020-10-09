package digest

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/pkg/errors"
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
		// There was a bug that caused digests that are created with a leading zero to be stored
		// in the DB without the leading zero. This bug has been fixed but these digests need to be
		// accounted for by appending the missing zero back onto the start of the digest.
		if len(spl[1]) == 95 {
			spl[1] = fmt.Sprintf("0%s", spl[1])
		} else if len(spl[1]) != 96 {
			return nil, fmt.Errorf("hash length is %d, expected length 96", len(spl[1]))
		}

		algorithm = SHA384
	default:
		return nil, errors.New("unsupported format %q")
	}

	sum, err := hex.DecodeString(spl[1])
	if err != nil {
		return nil, errors.Wrap(err, "converting string sum to hex failed")
	}

	return &Digest{
		Sum:       sum,
		Algorithm: algorithm,
	}, nil
}
