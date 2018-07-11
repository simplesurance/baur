package baur

// SrcResolver is an interface to resolve abstract paths like file glob paths to
// concrete values (files)
type SrcResolver interface {
	Resolve() (uri []string, err error)
}
