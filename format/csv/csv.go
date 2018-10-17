package csv

import (
	"encoding/csv"
	"fmt"
	"io"
)

// Formatter converts Rows into CSV format.
type Formatter struct {
	out       io.Writer
	csvWriter *csv.Writer
}

// New returns a new tabwriter, if headers is not empty it's written as first
// row to the output
func New(headers []string, out io.Writer) *Formatter {
	f := Formatter{
		out:       out,
		csvWriter: csv.NewWriter(out),
	}

	if len(headers) > 0 {
		_ = f.writeHeader(headers)
	}

	return &f
}

func (f *Formatter) writeHeader(headers []string) error {
	return f.csvWriter.Write(headers)
}

// WriteRow writes a row to the csvwriter buffer
func (f *Formatter) WriteRow(row []interface{}) error {
	var str []string

	for _, col := range row {
		str = append(str, fmt.Sprintf("%v", col))
	}

	return f.csvWriter.Write(str)
}

// Flush flushes the csvwriter buffer to it's output
func (f *Formatter) Flush() error {
	f.csvWriter.Flush()

	return f.csvWriter.Error()
}
