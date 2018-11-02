package resolve

// Resolver specifies an interface to resolve abstract file specifications like
// glob paths or go packages to file paths
type Resolver interface {
	Resolve() ([]string, error)
}
