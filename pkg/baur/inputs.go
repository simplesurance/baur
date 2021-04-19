package baur

import (
	"fmt"

	"github.com/simplesurance/baur/v2/internal/digest"
	"github.com/simplesurance/baur/v2/internal/digest/sha384"
)

// Inputs are resolved Inputs of a task.
type Inputs struct {
	inputs []Input
	digest *digest.Digest
}

// NewInputs returns an new Inputs
func NewInputs(in []Input) *Inputs {
	return &Inputs{inputs: in}
}

// Inputs returns all stored Inputs
func (in *Inputs) Inputs() []Input {
	return in.inputs
}

// Add adds elements in inputs to in and returns *in
func (in *Inputs) Add(inputs []Input) *Inputs {
	if len(inputs) == 0 {
		return in
	}

	in.inputs = append(in.inputs, inputs...)
	in.digest = nil

	return in
}

// Digest returns a summarized digest over all Inputs.
// On the first call the digest is calculated, on subsequent calls the stored digest is returned.
func (in *Inputs) Digest() (*digest.Digest, error) {
	if in.digest != nil {
		return in.digest, nil
	}

	digests := make([]*digest.Digest, len(in.inputs))

	for i, input := range in.inputs {
		fdigest, err := input.Digest()
		if err != nil {
			return nil, fmt.Errorf("calculating digest for %q failed: %w", input, err)
		}

		digests[i] = fdigest
	}

	totalDigest, err := sha384.Sum(digests)
	if err != nil {
		return nil, err
	}

	in.digest = totalDigest

	return in.digest, nil
}
