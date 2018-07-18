package sha384

import (
	"bytes"
	"crypto/sha512"
	"fmt"
	"hash"
	"io"
	"math/big"
	"os"
	"sort"

	"github.com/pkg/errors"
	"github.com/simplesurance/baur/digest"
)

// File returns the SHA256 digest of the file
func File(path string) (*digest.Digest, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrap(err, "opening file failed")
	}

	defer f.Close()

	h := sha512.New384()
	if _, err := io.Copy(h, f); err != nil {
		return nil, errors.Wrap(err, "reading file failed")
	}

	return HashToDigest(h)
}

// HashToDigest returns a SHA384 digest for a hash.Hash
func HashToDigest(h hash.Hash) (*digest.Digest, error) {
	sum := big.Int{}
	_, err := fmt.Sscan(fmt.Sprintf("0x%x", (h.Sum(nil))), &sum)
	if err != nil {
		return nil, errors.Wrap(err, "converting digest to big int failed")
	}

	return &digest.Digest{
		Algorithm: digest.SHA384,
		Sum:       sum,
	}, nil
}

// Bytes hashes the byte slice
func Bytes(b []byte) (*digest.Digest, error) {
	h := sha512.New384()
	_, err := h.Write(b)
	if err != nil {
		return nil, errors.Wrap(err, "writing to hash stream failed")
	}

	return HashToDigest(h)
}

// Sum aggregates multiple digests to a single SHA384 one
func Sum(digests []*digest.Digest) (*digest.Digest, error) {
	buf := bytes.Buffer{}

	sort.Slice(digests, func(i, j int) bool {
		if digests[i].Algorithm < digests[j].Algorithm {
			return true
		}

		if digests[i].Algorithm > digests[j].Algorithm {
			return false
		}

		return digests[i].Sum.Cmp(&digests[j].Sum) == -1
	})

	for _, d := range digests {
		buf.WriteString(d.String())
	}

	return Bytes(buf.Bytes())

}
