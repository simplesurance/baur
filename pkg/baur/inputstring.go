package baur

import (
	"fmt"

	"github.com/simplesurance/baur/v3/internal/digest"
	"github.com/simplesurance/baur/v3/internal/digest/sha384"
)

// InputString represents a string
type InputString struct {
	value  string
	digest *digest.Digest
}

// NewInputString returns a new InputString
func NewInputString(val string) *InputString {
	return &InputString{value: val}
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

// String returns it's string representation (string:VAL)
func (i *InputString) String() string {
	return fmt.Sprintf("string:%s", i.value)
}

// Value returns the string that the input represents.
func (i *InputString) Value() string {
	return i.value
}

// CalcDigest calculates the digest of the string, saves it and returns it.
func (i *InputString) calcDigest() (*digest.Digest, error) {
	sha := sha384.New()

	err := sha.AddBytes([]byte(i.value))
	if err != nil {
		return nil, err
	}

	i.digest = sha.Digest()

	return i.digest, nil
}

// AsInputStrings returns InputStrings for all elements in strs.
func AsInputStrings(strs ...string) []Input {
	result := make([]Input, 0, len(strs))

	for _, s := range strs {
		result = append(result, NewInputString(s))
	}

	return result
}
