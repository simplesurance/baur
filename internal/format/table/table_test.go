package table

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatterAlignsColumns(t *testing.T) {
	var buf bytes.Buffer
	f := New([]string{"NAME", "VALUE"}, &buf)

	require.NoError(t, f.WriteRow("a", 1))
	require.NoError(t, f.WriteRow("longer", nil))
	require.NoError(t, f.Flush())

	// column padding is 8 spaces after the longest cell
	assert.Equal(t,
		"NAME          VALUE\n"+
			"a             1\n"+
			"longer        \n",
		buf.String(),
	)
}

func TestFormatterWithoutHeaders(t *testing.T) {
	var buf bytes.Buffer
	f := New(nil, &buf)

	require.NoError(t, f.WriteRow("a", "b"))
	require.NoError(t, f.Flush())

	assert.Equal(t, "a        b\n", buf.String())
}

func TestWriteRowWithoutValues(t *testing.T) {
	var buf bytes.Buffer
	f := New(nil, &buf)

	require.NoError(t, f.WriteRow())
	require.NoError(t, f.Flush())

	assert.Equal(t, "\n", buf.String())
}
