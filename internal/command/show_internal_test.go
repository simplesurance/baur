package command

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/simplesurance/baur/v2/internal/format/table"
)

func TestMustWriteStringSliceRows(t *testing.T) {
	var buf bytes.Buffer
	formatter := table.New(nil, &buf)

	const hdr = "thehdr"
	const hdrIndent = "      " // " " * len(hdr)
	const indentlvl = 2
	const indentStr = "        "
	elems := []string{"one", "2", "three"}
	mustWriteStringSliceRows(formatter, hdr, indentlvl, elems)
	formatter.Flush()

	expectedOut := ("" +
		indentStr + indentStr + hdr + indentStr + elems[0] + ", " + "\n" +
		hdrIndent + indentStr + indentStr + indentStr + elems[1] + ", " + "\n" +
		hdrIndent + indentStr + indentStr + indentStr + elems[2] + "\n" +
		"")
	assert.Equal(t, expectedOut, buf.String())
}
