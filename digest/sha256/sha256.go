package sha256

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"

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

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return nil, errors.Wrap(err, "reading file failed")
	}

	return &digest.Digest{
		Algorithm: digest.SHA256,
		Sum:       fmt.Sprintf("%x", (h.Sum(nil))),
	}, nil
}
