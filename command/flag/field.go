package flag

import (
	"errors"
	"fmt"
	"strings"
)

// FieldSep specifies the separator when passing multiple values to the flag
const FieldSep = ","

// Fields is a commandline flag that allows to enable one or more boolean
// fields by passing them in a format like "-o showbuilds,Showdigests".
// The passed values are case-insensitive.
// By default, if Set() is not called the value for all fields is enabled/true.
// After Set() was called only the passed fields are enabled.
type Fields struct {
	fields map[string]bool
}

// NewFields returns a new flag that supports the passed fields.
func NewFields(values []string) *Fields {
	f := Fields{
		fields: map[string]bool{},
	}

	for _, v := range values {
		f.fields[strings.ToLower(v)] = true

	}

	return &f
}

// String returns the default value in the usage output
func (f *Fields) String() string {
	return ""
}

// ValidValues returns the values that the flag accepts
func (f *Fields) ValidValues() string {
	vals := make([]string, 0, len(f.fields))

	for k := range f.fields {
		vals = append(vals, k)
	}

	return strings.Join(vals, FieldSep+" ")
}

// Set parses a list oo.fields, fields contained in the string are set to
// true, all others are set to fault. If an unrecognized option value is passed
// a error is returned
func (f *Fields) Set(val string) error {
	substr := strings.Split(val, FieldSep)
	if len(substr) == 0 {
		return errors.New("format must be " + f.Type())
	}

	for opt := range f.fields {
		f.fields[opt] = false
	}

	for _, str := range substr {
		fieldStr := strings.TrimSpace(strings.ToLower(str))

		_, exist := f.fields[fieldStr]
		if !exist {
			return errors.New("<FIELD> must be one of " + f.ValidValues())
		}

		f.fields[fieldStr] = true
	}

	return nil
}

// Type returns the format description
func (f *Fields) Type() string {
	return "<FIELD>[,<FIELD>]..."
}

// Usage returns a usage description, important parts are passed through
// highlightFn
func (f *Fields) Usage(highlightFn func(a ...interface{}) string) string {
	fields := make([]string, 0, len(f.fields))

	for k := range f.fields {
		fields = append(fields, highlightFn(k))
	}

	return strings.TrimSpace(fmt.Sprintf(`
Output only the specified fields
Format: %s
where %s is one of: %s`,
		highlightFn(f.Type()),
		highlightFn("FIELD"), strings.Join(fields, ", ")))
}

// IsSet returns true if the field is one of the valid values for this flag and
// either Set() was never called or Set() was called with a value containing the
// field name
func (f *Fields) IsSet(field string) bool {
	set, exist := f.fields[strings.ToLower(field)]
	if !exist {
		return false
	}

	return set
}
