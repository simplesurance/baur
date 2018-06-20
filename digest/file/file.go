package file

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"

	"github.com/pkg/errors"
	"github.com/simplesurance/baur/digest"
)

// SHA256Digest returns the SHA256 Checksum of the file
func SHA256Digest(path string) (*digest.Digest, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return nil, errors.Wrap(err, "reading file failed")
	}

	return &digest.Digest{
		Sum:  fmt.Sprintf("%x", (h.Sum(nil))),
		Type: digest.SHA256,
	}, nil
}
