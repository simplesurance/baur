package baur

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiffTypeString(t *testing.T) {
	assert.Equal(t, "D", DigestMismatch.String())
	assert.Equal(t, "-", Removed.String())
	assert.Equal(t, "+", Added.String())
}

func TestDiffInputs(t *testing.T) {
	a := NewInputs([]Input{
		NewInputString("same"),
		NewInputEnvVar("CHANGED", "1"),
		NewInputEnvVar("ONLY_A", "x"),
	})
	b := NewInputs([]Input{
		NewInputString("same"),
		NewInputEnvVar("CHANGED", "2"),
		NewInputEnvVar("ONLY_B", "y"),
	})

	diffs, err := DiffInputs(a, b)
	require.NoError(t, err)
	require.Len(t, diffs, 3)

	// sorted by path
	assert.Equal(t, "$CHANGED", diffs[0].Path)
	assert.Equal(t, DigestMismatch, diffs[0].State)
	assert.NotEmpty(t, diffs[0].Digest1)
	assert.NotEmpty(t, diffs[0].Digest2)
	assert.NotEqual(t, diffs[0].Digest1, diffs[0].Digest2)

	assert.Equal(t, "$ONLY_A", diffs[1].Path)
	assert.Equal(t, Removed, diffs[1].State)
	assert.NotEmpty(t, diffs[1].Digest1)
	assert.Empty(t, diffs[1].Digest2)

	assert.Equal(t, "$ONLY_B", diffs[2].Path)
	assert.Equal(t, Added, diffs[2].State)
	assert.Empty(t, diffs[2].Digest1)
	assert.NotEmpty(t, diffs[2].Digest2)
}

func TestDiffInputsIdenticalSetsHaveNoDiff(t *testing.T) {
	mk := func() *Inputs {
		return NewInputs([]Input{NewInputString("a"), NewInputEnvVar("X", "1")})
	}

	diffs, err := DiffInputs(mk(), mk())
	require.NoError(t, err)
	assert.Empty(t, diffs)
}

func TestDiffInputsEmpty(t *testing.T) {
	diffs, err := DiffInputs(NewInputs(nil), NewInputs(nil))
	require.NoError(t, err)
	assert.Empty(t, diffs)
}

func TestDiffInputsReturnsErrorIfDigestFails(t *testing.T) {
	bad := NewInputs([]Input{NewInputFile("/abs/f", "f", false)})
	ok := NewInputs(nil)

	_, err := DiffInputs(bad, ok)
	require.Error(t, err)

	_, err = DiffInputs(ok, bad)
	require.Error(t, err)
}
