package postgres

import (
	"fmt"
	"strings"
)

type queryError struct {
	Query     string
	Arguments []interface{}
	Err       error
}

func (e *queryError) Unwrap() error {
	return e.Err
}

func (e *queryError) Error() string {
	return fmt.Sprintf("%s\nquery:\n---\n%s\n---\narguments: %s",
		e.Err,
		strings.TrimSpace(e.Query),
		strArgList(e.Arguments...),
	)
}

func newQueryError(query string, err error, args ...interface{}) *queryError {
	return &queryError{
		Query:     query,
		Arguments: args,
		Err:       err,
	}
}
