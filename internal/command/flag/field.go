package flag

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// FieldSep specifies the separator when passing multiple values to the flag
const FieldSep = ","

// Fields is a flag that accept a comma-separated list of field names.
// fields by passing them in a format like "-f id,duration,path".
// The passed values are case-insensitive.
type Fields struct {
	supportedFields map[string]struct{}
	Fields          []string
}

// MustNewFields returns a new flag that supports the passed fields.
// The default value of the flag is specified by defaultFields.
// If an element is in defaultFields but not in supportedFields the function
// panics.
func MustNewFields(supportedFields, defaultFields []string) *Fields {
	res := Fields{
		supportedFields: make(map[string]struct{}, len(supportedFields)),
		Fields:          make([]string, 0, len(supportedFields)),
	}

	for _, f := range supportedFields {
		lf := strings.ToLower(f)

		res.supportedFields[lf] = struct{}{}
	}

	for _, f := range defaultFields {
		lf := strings.ToLower(f)

		if _, exist := res.supportedFields[lf]; !exist {
			panic(fmt.Sprintf("%q listed in defaultFields but not in supportedFields", f))
		}

		res.Fields = append(res.Fields, lf)
	}

	return &res
}

// String returns the list of enabled fields.
// It is used to show the default  value in the usage output.
func (f *Fields) String() string {
	// the returned list does not contains spaces, it must in the same format
	// that the parameter accepts as value
	return strings.Join(f.Fields, FieldSep)
}

// ValidValues returns the values that the flag accepts
func (f *Fields) ValidValues() string {
	vals := make([]string, 0, len(f.supportedFields))

	for k := range f.supportedFields {
		vals = append(vals, k)
	}

	sort.Strings(vals)

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

	setFields := make([]string, 0, len(substr))

	for _, str := range substr {
		fieldStr := strings.TrimSpace(strings.ToLower(str))

		if _, exist := f.supportedFields[fieldStr]; !exist {
			return errors.New("FIELD must be one of " + f.ValidValues())
		}

		setFields = append(setFields, fieldStr)
	}

	f.Fields = setFields

	return nil
}

// Type returns the format description
func (f *Fields) Type() string {
	return "FIELD[,FIELD]..."
}

// Usage returns a usage description, important parts are passed through
// highlightFn
func (f *Fields) Usage(highlightFn func(a ...any) string) string {
	fields := make([]string, 0, len(f.supportedFields))
	for k := range f.supportedFields {
		fields = append(fields, k)
	}

	sort.Strings(fields)

	for i, f := range fields {
		fields[i] = highlightFn(f)
	}

	return fmt.Sprintf(`Specify the printed fields and their order:
Format: %s
where %s is one of: %s
`,
		highlightFn(f.Type()),
		highlightFn("FIELD"), strings.Join(fields, ", "),
	)
}
