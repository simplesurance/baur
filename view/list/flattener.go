package list

type StringHighlighterFunc func(a ...interface{}) string

// FlattenerFunc is a function that converts a List to string.
// If Quiet is specified, a short version should be returned, typically the first column only.
type FlattenerFunc func(list List, hi StringHighlighterFunc, quiet bool) (string, error)

func (f FlattenerFunc) FlattenList(list List, hi StringHighlighterFunc, quiet bool) (string, error) {
	return f(list, hi, quiet)
}
