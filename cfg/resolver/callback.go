package resolver

import (
	"fmt"
	"strings"
)

// CallbackReplacement replaces Old with the string that NewFunc returns.
type CallbackReplacement struct {
	Old     string
	NewFunc func() (string, error)
}

func (c *CallbackReplacement) Resolve(in string) (string, error) {
	// only run NewFunc() if necessary to prevent that Resolve() returns an
	// error because NewFunc() failed, despite the string does not contain
	// the substring.
	if !strings.Contains(in, c.Old) {
		return in, nil
	}

	new, err := c.NewFunc()
	if err != nil {
		return "", fmt.Errorf("could not replace cfg variable %s: %w", in, err)
	}

	return strings.Replace(in, c.Old, new, -1), nil
}
