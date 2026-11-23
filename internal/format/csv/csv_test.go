package csv

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatterWritesHeaderAndRows(t *testing.T) {
	var buf bytes.Buffer
	f := New([]string{"name", "value"}, &buf)

	require.NoError(t, f.WriteRow("a", 1))
	require.NoError(t, f.WriteRow("b", nil))
	require.NoError(t, f.WriteRow("with,comma", "with \"quote\""))
	require.NoError(t, f.Flush())

	assert.Equal(t, "name,value\na,1\nb,\n\"with,comma\",\"with \"\"quote\"\"\"\n", buf.String())
}

func TestFormatterWithoutHeaders(t *testing.T) {
	var buf bytes.Buffer
	f := New(nil, &buf)

	require.NoError(t, f.WriteRow("a", true, 1.5))
	require.NoError(t, f.Flush())

	assert.Equal(t, "a,true,1.5\n", buf.String())
}

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

func TestFlushReturnsWriteError(t *testing.T) {
	f := New(nil, failWriter{})

	_ = f.WriteRow("a")

	require.Error(t, f.Flush())
}
