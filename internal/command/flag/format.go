package flag

const (
	FormatFlagName = "format"
	FormatCSV      = "csv"
	FormatJSON     = "json"
	FormatPlain    = "plain"
)

type Format struct {
	Val string
}

func NewFormatFlag() *OneOf {
	return NewOneOfFlag(
		FormatFlagName,
		FormatPlain,
		"output format",
		FormatCSV, FormatJSON, FormatPlain,
	)
}
