package flag

import (
	"errors"
	"fmt"
	"strings"

	"github.com/simplesurance/baur/v3/pkg/storage"
)

// Sort is a commandline flag to specify via which field and in which order
// output should be sorted
type Sort struct {
	Value       storage.Sorter
	validFields map[string]storage.Field
}

// NewSort returns a Sort flag.
// The keys in the validFields maps are accepted as value for
// Fields and map to the storage.Field values.
func NewSort(validFields map[string]storage.Field) *Sort {
	return &Sort{
		validFields: validFields,
	}
}

// String returns the default value in the usage output
func (s *Sort) String() string {
	return ""
}

// Set parses the passed string and sets the Sort
func (s *Sort) Set(sortStr string) error {
	var err error

	pieces := strings.Split(sortStr, "-")
	if len(pieces) != 2 {
		return fmt.Errorf("format must be %s", s.Type())
	}

	field, exist := s.validFields[strings.ToLower(pieces[0])]
	if !exist {
		return errors.New("format must be: " + s.Type())
	}

	order, err := storage.OrderFromStr(pieces[1])
	if err != nil {
		return err
	}

	s.Value.Field = field
	s.Value.Order = order

	return nil
}

// Type returns the format description
func (s *Sort) Type() string {
	return "FIELD-ORDER"
}

// Usage returns a usage description, important parts are passed through
// highlightFn
func (s *Sort) Usage(highlightFn func(a ...interface{}) string) string {
	fields := make([]string, 0, len(s.validFields))

	for k := range s.validFields {
		fields = append(fields, highlightFn(k))
	}

	return strings.TrimSpace(fmt.Sprintf(`
Sort the list by a specific field.
Format: %s
where %s is one of: %s,
and %s one of: %s, %s`,
		highlightFn(s.Type()),
		highlightFn("FIELD"), strings.Join(fields, ", "),
		highlightFn("ORDER"), highlightFn(storage.OrderAsc.String()), highlightFn(storage.OrderDesc.String())))
}
