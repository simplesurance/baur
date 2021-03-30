package cfg

// Resolver is an interface for replacing substrings with a special meaning in strings.
type Resolver interface {
	Resolve(string) (string, error)
}
