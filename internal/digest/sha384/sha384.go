package sha384

import (
	"bytes"
	"crypto/sha512"
	"fmt"
	stdhash "hash"
	"io"
	"os"
	"sort"

	"github.com/simplesurance/baur/v3/internal/digest"
)

// Hash offers an interface to add data for computing a digest
type Hash struct {
	hash stdhash.Hash
}

// New returns a  sha384.Hash to compute a digest
func New() *Hash {
	return &Hash{hash: sha512.New384()}
}

// AddFile reads a file and adds it to the hash
func (h *Hash) AddFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("opening file failed: %w", err)
	}

	defer f.Close()

	if _, err := io.Copy(h.hash, f); err != nil {
		return fmt.Errorf("reading file failed: %w", err)
	}

	return nil
}

// Digest returns the digest of the hash
func (h *Hash) Digest() *digest.Digest {
	sum := h.hash.Sum(nil)

	return &digest.Digest{
		Algorithm: digest.SHA384,
		Sum:       sum,
	}
}

// AddBytes add bytes to the hash
func (h *Hash) AddBytes(b []byte) error {
	_, err := h.hash.Write(b)
	if err != nil {
		return fmt.Errorf("writing to hash stream failed: %w", err)
	}

	return nil
}

// Sum aggregates multiple digests to a single SHA384 digest
func Sum(digests []*digest.Digest) (*digest.Digest, error) {
	hash := New()
	buf := bytes.Buffer{}

	sort.Slice(digests, func(i, j int) bool {
		if digests[i].Algorithm < digests[j].Algorithm {
			return true
		}

		if digests[i].Algorithm > digests[j].Algorithm {
			return false
		}

		return bytes.Compare(digests[i].Sum, digests[j].Sum) == -1
	})

	for _, d := range digests {
		buf.WriteString(d.String())
	}

	if err := hash.AddBytes(buf.Bytes()); err != nil {
		return nil, err
	}

	return hash.Digest(), nil
}
