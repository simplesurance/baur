package jsonformat

import (
	"encoding/json"
	"fmt"
	"io"
)

type Formatter struct {
	fieldNames []string
	data       []map[string]any
	w          io.Writer
}

func New(fieldNames []string, w io.Writer) *Formatter {
	return &Formatter{
		fieldNames: fieldNames,
		w:          w,
	}
}

// WriteRow adds a new entry to the internal map.
// Vals must be in the same order then the fieldNames that were passed when
// NewFormatter was called.
func (f *Formatter) WriteRow(vals ...any) error {
	if len(vals) != len(f.fieldNames) {
		return fmt.Errorf("got %d values, having %d headers, expecting the same amount", len(vals), len(f.fieldNames))
	}

	entry := make(map[string]any, len(vals))
	for i, v := range vals {
		entry[f.fieldNames[i]] = v
	}

	f.data = append(f.data, entry)

	return nil
}

// Flush writes the JSON encoding to the io.Writer.
// On success the internal map is cleared.
func (f *Formatter) Flush() error {
	enc := json.NewEncoder(f.w)
	enc.SetIndent("", "  ")

	if err := enc.Encode(f.data); err != nil {
		return err
	}

	clear(f.data)
	return nil
}
