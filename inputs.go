package baur

import (
	"fmt"

	"github.com/simplesurance/baur/v1/internal/digest"
	"github.com/simplesurance/baur/v1/internal/digest/sha384"
)

// Inputs are resolved Inputs of a task.
type Inputs struct {
	Files       []*Inputfile
	inputString *InputString
	digest      *digest.Digest
}

// NewInputs returns a new Inputs
func NewInputs(files []*Inputfile) *Inputs {
	return &Inputs{Files: files}
}

// SetInputString sets a string value as an *InputString
func (in *Inputs) SetInputString(inputStr string) {
	in.inputString = NewInputString(inputStr)
	in.digest = nil
}

// GetInputString returns an *InputString set via SetInputString
func (in *Inputs) GetInputString() *InputString {
	return in.inputString
}

// Digest returns a summarized digest over all Inputs.
// On the first call the digest is calculated, on subsequent calls the stored digest is returned.
func (in *Inputs) Digest() (*digest.Digest, error) {
	if in.digest != nil {
		return in.digest, nil
	}

	digests := make([]*digest.Digest, len(in.Files))

	for i, file := range in.Files {
		fdigest, err := file.Digest()
		if err != nil {
			return nil, fmt.Errorf("calculating digest for %q failed: %w", file.Path(), err)
		}

		digests[i] = fdigest
	}

	if in.inputString.Exists() {
		idigest, err := in.inputString.Digest()
		if err != nil {
			return nil, fmt.Errorf("calculating digest for input string %q failed: %w", in.inputString.Value, err)
		}

		digests = append(digests, idigest)
	}

	totalDigest, err := sha384.Sum(digests)
	if err != nil {
		return nil, err
	}

	in.digest = totalDigest

	return in.digest, nil
}
