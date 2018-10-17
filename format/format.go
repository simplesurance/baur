// Package format outputs data in formatted table structures
package format

// Formatter is an interface for formatters
type Formatter interface {
	WriteRow(Row []interface{}) error
	Flush() error
}
