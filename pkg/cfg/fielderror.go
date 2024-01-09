package cfg

import (
	"errors"
	"fmt"
	"strings"
)

// fieldError describes an error related to an element in a configuration struct.
type fieldError struct {
	elementPath []string
	err         error
}

// newFieldError creates a new FieldError with the given error message and ElementPath.
func newFieldError(msg string, path ...string) *fieldError {
	return &fieldError{
		err:         errors.New(msg),
		elementPath: path,
	}
}

// fieldErrorWrap returns a new FieldError thats wraps the passed err, if err
// is not of type FieldError.
// If it is of type FieldError, the passed paths are prepended to it's
// ElementPath and err is returned.
func fieldErrorWrap(err error, path ...string) error {
	var fErr *fieldError
	if errors.As(err, &fErr) {
		fErr.elementPath = append(path, fErr.elementPath...)
		return err
	}

	return &fieldError{
		elementPath: path,
		err:         err,
	}
}

func (f *fieldError) Error() string {
	return fmt.Sprintf("%s: %s", strings.Join(f.elementPath, "."), f.err)
}

func (f *fieldError) Unwrap() error {
	if err := errors.Unwrap(f.err); err != nil {
		return err
	}

	return f.err
}
