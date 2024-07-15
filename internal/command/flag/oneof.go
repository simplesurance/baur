package flag

import (
	"fmt"
	"slices"
	"strings"
	"unicode"

	"github.com/simplesurance/baur/v5/internal/set"

	"github.com/spf13/cobra"
)

// OneOf is a command line flag that accepts one of multiple possible values.
type OneOf struct {
	Val       string
	supported set.Set[string]
	flagName  string
	usage     string
}

func NewOneOfFlag(flagName, defaultVal, usage string, supportedVals ...string) *OneOf {
	for _, v := range supportedVals {
		if !isLower(v) {
			panic(fmt.Sprintf("oneOf flag values must be lowercase, got: %q", v))
		}
	}

	return &OneOf{
		flagName:  flagName,
		Val:       defaultVal,
		supported: set.From(supportedVals),
		usage:     usage,
	}
}

func (f *OneOf) Set(val string) error {
	sl := strings.ToLower(val)
	if !f.supported.Contains(sl) {
		sortedVals := f.supported.Slice()
		slices.Sort(sortedVals)
		return fmt.Errorf("%s must be one of: %s",
			f.flagName, strings.Join(sortedVals, ", "))
	}

	f.Val = sl
	return nil
}

func (f *OneOf) Value() string {
	return f.Val
}

func (f *OneOf) String() string {
	return f.Val
}

func (f *OneOf) Type() string {
	return strings.ToUpper(f.flagName)
}

func (f *OneOf) Usage(highlightFn func(a ...any) string) string {
	var res strings.Builder
	res.WriteString(f.usage)
	res.WriteRune('\n')
	res.WriteString("one of: ")

	var cnt int
	for v := range f.supported {
		res.WriteString(highlightFn(v))
		cnt++
		if len(f.supported) > cnt {
			res.WriteString(", ")
		}
	}

	return res.String()
}

func (f *OneOf) RegisterFlagCompletion(cmd *cobra.Command) error {
	return cmd.RegisterFlagCompletionFunc(f.flagName, func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		return f.supported.Slice(), cobra.ShellCompDirectiveDefault
	})
}

func isLower(s string) bool {
	for _, r := range s {
		if !unicode.IsLower(r) {
			return false
		}
	}
	return true
}
