package exec

import "fmt"

// ExitCodeError is returned from Run() when a command exited with a code != 0.
type ExitCodeError struct {
	*Result
}

// Error returns the error description.
func (e ExitCodeError) Error() string {
	return fmt.Sprintf("exec: running '%s' in directory '%s' exited with code %d, expected 0, output: '%s'",
		e.Command, e.Dir, e.ExitCode, e.Output)
}
