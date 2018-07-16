package sha384

import (
	"crypto/sha512"
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

	h := sha512.New384()
	if _, err := io.Copy(h, f); err != nil {
		return nil, errors.Wrap(err, "reading file failed")
	}

	return &digest.Digest{
		Algorithm: digest.SHA384,
		Sum:       fmt.Sprintf("%x", (h.Sum(nil))),
	}, nil
}
