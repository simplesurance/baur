package command

import (
	"fmt"

	"github.com/simplesurance/baur/v2/internal/digest"
	"github.com/simplesurance/baur/v2/pkg/baur"
	"github.com/simplesurance/baur/v2/pkg/storage"
)

type storageInputFile struct {
	*storage.InputFile
}

func (i *storageInputFile) Digest() (*digest.Digest, error) {
	return digest.FromString(i.InputFile.Digest)
}

func (i *storageInputFile) String() string {
	return i.InputFile.Path
}

type storageInputString struct {
	*storage.InputString
}

func (i *storageInputString) Digest() (*digest.Digest, error) {
	return digest.FromString(i.InputString.Digest)
}

func (i *storageInputString) String() string {
	return fmt.Sprintf("string:%s", i.InputString.String)
}

func toBaurInputs(inputs *storage.Inputs) []baur.Input {
	result := make([]baur.Input, 0, len(inputs.Files)+len(inputs.Strings))

	for _, in := range inputs.Files {
		result = append(result, &storageInputFile{InputFile: in})
	}

	for _, in := range inputs.Strings {
		result = append(result, &storageInputString{InputString: in})
	}

	return result
}
