package baur

// BuildInputPathResolver is an interface to resolve abstract paths like file glob paths to
// concrete values (files)
type BuildInputPathResolver interface {
	Resolve() ([]BuildInput, error)
	// Type returns the type of resolver
	Type() string
	String() string
}
