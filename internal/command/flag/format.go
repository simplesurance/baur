package flag

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

const (
	FormatCSV   = "csv"
	FormatJSON  = "json"
	FormatPlain = "plain"
)

const formatUsage = "one of: " + FormatCSV + ", " + FormatJSON + ", " + FormatPlain

type Format struct {
	Val string
}

func NewFormatFlag() *Format {
	return &Format{Val: FormatPlain}
}

// Set parses the passed string and sets the SortFlagValue
func (f *Format) Set(val string) error {
	switch v := strings.ToLower(val); v {
	case FormatPlain, FormatJSON, FormatCSV:
		f.Val = v
	default:
		return errors.New("format must be " + formatUsage)
	}

	return nil
}

func (f *Format) String() string {
	return f.Val
}

func (f *Format) Type() string {
	return "FORMAT"
}

func (f *Format) Usage(highlightFn func(a ...any) string) string {
	return fmt.Sprintf("output format\none of: %s, %s, %s",
		highlightFn(FormatCSV), highlightFn(FormatJSON), highlightFn(FormatPlain),
	)
}

func (f *Format) RegisterFlagCompletion(cmd *cobra.Command) error {
	return cmd.RegisterFlagCompletionFunc("format", func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		return []string{FormatCSV, FormatJSON, FormatPlain}, cobra.ShellCompDirectiveDefault
	})
}
