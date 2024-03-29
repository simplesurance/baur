package command

import (
	"fmt"

	"github.com/simplesurance/baur/v3/internal/digest"
	"github.com/simplesurance/baur/v3/pkg/baur"
	"github.com/simplesurance/baur/v3/pkg/storage"
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

type storageInputEnvVar struct {
	*storage.InputEnvVar
}

func (i *storageInputEnvVar) String() string {
	return "$" + i.InputEnvVar.Name
}

func (i *storageInputEnvVar) Digest() (*digest.Digest, error) {
	return digest.FromString(i.InputEnvVar.Digest)
}

func toBaurInputs(inputs *storage.Inputs) []baur.Input {
	result := make([]baur.Input, 0, len(inputs.Files)+len(inputs.Strings))

	for _, in := range inputs.Files {
		result = append(result, &storageInputFile{InputFile: in})
	}

	for _, in := range inputs.Strings {
		result = append(result, &storageInputString{InputString: in})
	}

	for _, in := range inputs.EnvironmentVariables {
		result = append(result, &storageInputEnvVar{InputEnvVar: in})
	}

	return result
}
