package resolver

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCallbackReplacementSuccessful(t *testing.T) {
	old := "$yo"
	new := "no"

	c := CallbackReplacement{
		Old:     old,
		NewFunc: func() (string, error) { return new, nil },
	}

	res, err := c.Resolve(old)
	require.NoError(t, err)
	assert.Equal(t, res, new)

}

func TestCallbackReplacementFails(t *testing.T) {
	old := "$yo"

	c := CallbackReplacement{
		Old:     old,
		NewFunc: func() (string, error) { return "", errors.New("fail") },
	}

	res, err := c.Resolve(old)
	require.Error(t, err)
	assert.Equal(t, res, "")
}
