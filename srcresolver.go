package baur

// SrcResolver is an interface to resolve abstract paths like file glob paths to
// concrete values (files)
// TODO rename it to BuildInputPath or similar?
type SrcResolver interface {
	Resolve() ([]BuildInput, error)
}
