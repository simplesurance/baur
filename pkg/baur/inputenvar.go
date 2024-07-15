package baur

import (
	"fmt"

	"github.com/simplesurance/baur/v5/internal/digest"
	"github.com/simplesurance/baur/v5/internal/digest/sha384"
)

// InputEnvVar represents an environment variable that is tracked as baur
// input.
type InputEnvVar struct {
	name   string
	value  string
	digest *digest.Digest
}

// NewInputEnvVar creates an InputEnvVar.
// InputEnvVar can not distinguish between empty and unset environment variables.
func NewInputEnvVar(name, value string) *InputEnvVar {
	return &InputEnvVar{name: name, value: value}
}

func (v *InputEnvVar) Digest() (*digest.Digest, error) {
	if v.digest != nil {
		return v.digest, nil
	}

	return v.calcDigest()
}

func (v *InputEnvVar) calcDigest() (*digest.Digest, error) {
	sha := sha384.New()

	hashStr := fmt.Sprintf("ENV: %s=%s", v.name, v.value)
	err := sha.AddBytes([]byte(hashStr))
	if err != nil {
		return nil, err
	}

	v.digest = sha.Digest()

	return v.digest, nil
}

// String returns the name of the environment variable prefixed with a "$".
func (v *InputEnvVar) String() string {
	return "$" + v.name
}

// Name returns the name of the environment variable.
func (v *InputEnvVar) Name() string {
	return v.name
}
