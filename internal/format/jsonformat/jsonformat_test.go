package jsonformat

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatterOutputsRowsAsJSONObjects(t *testing.T) {
	var buf bytes.Buffer
	f := New([]string{"name", "n"}, &buf)

	require.NoError(t, f.WriteRow("a", 1))
	require.NoError(t, f.WriteRow("b", nil))
	require.NoError(t, f.Flush())

	assert.JSONEq(t, `[{"name":"a","n":1},{"name":"b","n":null}]`, buf.String())
}

func TestWriteRowRejectsWrongValueCount(t *testing.T) {
	f := New([]string{"a", "b"}, &bytes.Buffer{})

	require.Error(t, f.WriteRow("only one"))
	require.Error(t, f.WriteRow(1, 2, 3))
}

func TestFlushWithoutRowsWritesNull(t *testing.T) {
	var buf bytes.Buffer
	f := New([]string{"a"}, &buf)

	require.NoError(t, f.Flush())

	assert.JSONEq(t, `null`, buf.String(), "characterization: no rows are encoded as null, not []")
}

// After Flush the rows are written and removed, a second Flush must not write them again.
func TestFlushClearsRows(t *testing.T) {
	var buf bytes.Buffer
	f := New([]string{"a"}, &buf)
	require.NoError(t, f.WriteRow("x"))
	require.NoError(t, f.Flush())
	buf.Reset()

	require.NoError(t, f.Flush())

	assert.JSONEq(t, `null`, buf.String())
}
