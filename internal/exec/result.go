package exec

import (
	"fmt"
	"strings"
)

// ExitCodeError is returned from Run() when a command exited with a code != 0.
type ExitCodeError struct {
	*Result
}

// Error returns the error description.
func (e ExitCodeError) Error() string {
	var result strings.Builder
	var stdoutExists bool

	result.WriteString(fmt.Sprintf("command exited with code %d, output:\n", e.ExitCode))

	if b := e.stdout.Bytes(); len(b) > 0 {
		result.WriteString("### stdout ###\n")
		result.WriteString(strings.TrimSpace(string(b)))
		result.WriteRune('\n')
		stdoutExists = true
	}

	if b := e.stderr.Bytes(); len(b) > 0 {
		if stdoutExists {
			result.WriteRune('\n')
		}
		result.WriteString("### stderr ###\n")
		result.WriteString(strings.TrimSpace(string(b)))
		result.WriteRune('\n')
	}

	return result.String()
}

// Result describes the result of a run Cmd.
type Result struct {
	Command  string
	Dir      string
	ExitCode int

	stdout *prefixSuffixSaver
	stderr *prefixSuffixSaver
}

// ExpectSuccess if the ExitCode in Result is not 0, the function returns an
// ExitCodeError for the execution.
func (r *Result) ExpectSuccess() error {
	if r.ExitCode == 0 {
		return nil
	}

	return ExitCodeError{Result: r}
}

type ResultOut struct {
	*Result
	CombinedOutput []byte
}

func (r *ResultOut) StrOutput() string {
	return string(r.CombinedOutput)
}
