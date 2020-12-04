package command

import (
	"github.com/simplesurance/baur/v1"
	"github.com/simplesurance/baur/v1/internal/digest"
	"github.com/simplesurance/baur/v1/storage"
)

type storageInput struct {
	input *storage.Input
}

func (i *storageInput) Digest() (*digest.Digest, error) {
	return digest.FromString(i.input.Digest)
}

func (i *storageInput) String() string {
	return i.input.URI
}

func toBaurInputs(inputs []*storage.Input) []baur.Input {
	result := make([]baur.Input, 0, len(inputs))

	for _, in := range inputs {
		result = append(result, &storageInput{input: in})
	}

	return result
}
