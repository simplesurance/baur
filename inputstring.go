package baur

import (
	"fmt"

	"github.com/simplesurance/baur/v1/internal/digest"
	"github.com/simplesurance/baur/v1/internal/digest/sha384"
)

// InputString represents a string
type InputString struct {
	value  string
	digest *digest.Digest
}

// CalcDigest calculates the digest of the string, saves it and returns it.
func (i *InputString) CalcDigest() (*digest.Digest, error) {
	sha := sha384.New()

	err := sha.AddBytes([]byte(i.value))
	if err != nil {
		return nil, err
	}

	i.digest = sha.Digest()

	return i.digest, nil
}

// Digest returns the previous calculated digest.
// If the digest wasn't calculated yet, CalcDigest() is called and it's return
// values are returned.
func (i *InputString) Digest() (*digest.Digest, error) {
	if i.digest != nil {
		return i.digest, nil
	}

	return i.CalcDigest()
}

// Exists returns whether an additional input string has been set
func (i *InputString) Exists() bool {
	return i.value != ""
}

// Value returns it's raw value
func (i *InputString) Value() string {
	return i.value
}

// String returns it's full string representation
func (i *InputString) String() string {
	return fmt.Sprintf("additional_string: %v", i.value)
}
