// Package format outputs data in formatted table structures
package format

// Row is a row of data
// The order of the elements in the Data slice must be the same then in the
// belonging Columns struct.
type Row struct {
	Data []interface{}
}

// Formatter is an interface for formatters
type Formatter interface {
	WriteRow(*Row) error
	Flush() error
}
