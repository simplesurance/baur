package resolver

import "strings"

// CallbackReplacement replaces Old with the string that NewFunc returns.
type CallbackReplacement struct {
	Old     string
	NewFunc func() (string, error)
}

func (c *CallbackReplacement) Resolve(in string) (string, error) {
	new, err := c.NewFunc()
	if err != nil {
		return "", err
	}

	return strings.Replace(in, c.Old, new, -1), nil
}
