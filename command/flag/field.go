package flag

import (
	"errors"
	"fmt"
	"strings"
)

// FieldSep specifies the separator when passing multiple values to the flag
const FieldSep = ","

// Fields is a commandline flag that allows to enable one or more boolean
// fields by passing them in a format like "-f id,duration,path".
// The passed values are case-insensitive.
type Fields struct {
	supportedFields map[string]struct{}
	Fields          []string
}

// NewFields returns a new flag that supports the passed fields.
func NewFields(fields []string) *Fields {
	res := Fields{
		supportedFields: map[string]struct{}{},
		Fields:          make([]string, 0, len(fields)),
	}

	for _, f := range fields {
		lf := strings.ToLower(f)

		res.supportedFields[lf] = struct{}{}
		res.Fields = append(res.Fields, lf)
	}

	return &res
}

// String returns the default value in the usage output
func (f *Fields) String() string {
	return fmt.Sprintf("\"%s\"", strings.Join(f.Fields, ", "))
}

// ValidValues returns the values that the flag accepts
func (f *Fields) ValidValues() string {
	vals := make([]string, 0, len(f.supportedFields))

	for k := range f.supportedFields {
		vals = append(vals, k)
	}

	return strings.Join(vals, FieldSep+" ")
}

// Set parses a list of fields, fields contained in the string are set to
// true, all others are set to fault. If an unrecognized option value is passed
// a error is returned
func (f *Fields) Set(val string) error {
	substr := strings.Split(val, FieldSep)
	if len(substr) == 0 {
		return errors.New("format must be " + f.Type())
	}

	var setFields []string

	for _, str := range substr {
		fieldStr := strings.TrimSpace(strings.ToLower(str))

		if _, exist := f.supportedFields[fieldStr]; !exist {
			return errors.New("<FIELD> must be one of " + f.ValidValues())
		}

		setFields = append(setFields, fieldStr)
	}

	f.Fields = setFields

	return nil
}

// Type returns the format description
func (f *Fields) Type() string {
	return "<FIELD>[,<FIELD>]..."
}

// Usage returns a usage description, important parts are passed through
// highlightFn
func (f *Fields) Usage(highlightFn func(a ...interface{}) string) string {
	fields := make([]string, 0, len(f.Fields))

	for _, f := range f.Fields {
		fields = append(fields, highlightFn(f))
	}

	return fmt.Sprintf(`Specify the printed fields and their order:
Format: %s
where %s is one of: %s
`,
		highlightFn(f.Type()),
		highlightFn("FIELD"), strings.Join(fields, ", "))
}
