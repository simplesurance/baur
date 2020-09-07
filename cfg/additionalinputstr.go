package cfg

import "github.com/simplesurance/baur/v1/cfg/resolver"

// AdditionalInputStr describes an additional string that is provided by command line
type AdditionalInputStr struct {
	Value string
}

func (a *AdditionalInputStr) Resolve(resolvers resolver.Resolver) error {
	var err error
	if a.Value, err = resolvers.Resolve(a.Value); err != nil {
		return FieldErrorWrap(err, "Value", a.Value)
	}

	return nil
}
