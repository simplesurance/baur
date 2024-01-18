// Package format outputs data in formatted table structures
package format

// Formatter is an interface for formatters
type Formatter interface {
	WriteRow(Row ...any) error
	Flush() error
}
