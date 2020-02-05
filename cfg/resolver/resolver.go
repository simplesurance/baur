package resolver

// Resolver is an interface for replacing substrings with a special meaning in strings.
type Resolver interface {
	Resolve(string) (string, error)
}

// List is a slice of Resolver.
type List []Resolver

// Resolve calls Resolve() on all resolvers in the List.
func (l List) Resolve(in string) (string, error) {
	for _, resolver := range l {
		var err error

		in, err = resolver.Resolve(in)
		if err != nil {
			return "", err
		}
	}

	return in, nil
}
