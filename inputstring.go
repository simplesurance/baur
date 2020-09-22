package baur

import (
	"fmt"

	"github.com/simplesurance/baur/v1/internal/digest"
	"github.com/simplesurance/baur/v1/internal/digest/sha384"
)

// InputString represents a string
type InputString struct {
	Value  string
	digest *digest.Digest
}

// NewInputString returns a new InputString
func NewInputString(value string) *InputString {
	return &InputString{Value: value}
}

// Digest returns the previous calculated digest.
// If the digest wasn't calculated yet, CalcDigest() is called and it's return
// values are returned.
func (i *InputString) Digest() (*digest.Digest, error) {
	if i.digest != nil {
		return i.digest, nil
	}

	return i.calcDigest()
}

// String returns it's string representation
func (i *InputString) String() string {
	return fmt.Sprintf("string:%s", i.Value)
}

// CalcDigest calculates the digest of the string, saves it and returns it.
func (i *InputString) calcDigest() (*digest.Digest, error) {
	sha := sha384.New()

	err := sha.AddBytes([]byte(i.Value))
	if err != nil {
		return nil, err
	}

	i.digest = sha.Digest()

	return i.digest, nil
}
