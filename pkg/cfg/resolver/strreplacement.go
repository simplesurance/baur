package resolver

import "strings"

// StrReplacement replaces the string Old with New.
type StrReplacement struct {
	Old string
	New string
}

func (s *StrReplacement) Resolve(in string) (string, error) {
	return strings.Replace(in, s.Old, s.New, -1), nil
}
