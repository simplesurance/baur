package cfg

import (
	"errors"
	"fmt"
	"strings"
)

// FieldError describes an error related to an element in a configuration struct.
type FieldError struct {
	elementPath []string
	err         error
}

// NewFieldError creates a new FieldError with the given error message and ElementPath.
func NewFieldError(msg string, path ...string) *FieldError {
	return &FieldError{
		err:         errors.New(msg),
		elementPath: path,
	}
}

// FieldErrorWrap returns a new FieldError thats wraps the passed err, if err
// is not of type FieldError.
// If it is of type FieldError, the passed paths are prepended to it's
// ElementPath and err is returned.
func FieldErrorWrap(err error, path ...string) error {
	valError, ok := err.(*FieldError)
	if ok {
		valError.elementPath = append(path, valError.elementPath...)
		return err
	}

	return &FieldError{
		elementPath: path,
		err:         err,
	}
}

func (f *FieldError) Error() string {
	return fmt.Sprintf("%s: %s", strings.Join(f.elementPath, "."), f.err)
}
