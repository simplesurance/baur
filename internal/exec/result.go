package exec

import (
	"os/exec"
	"strings"
)

// ExitCodeError is returned from Run() when a command exited with a code != 0.
type ExitCodeError struct {
	*Result
}

// Error returns the error description.
func (e *ExitCodeError) Error() string {
	var result strings.Builder
	var stdoutExists bool

	result.WriteString("execution failed: ")
	result.WriteString(e.ee.String())

	if len(e.stdout.Bytes()) == 0 && len(e.stderr.Bytes()) == 0 {
		return result.String()
	}

	result.WriteString(", output:\n")

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
	ee       *exec.ExitError
	success  bool

	stdout *prefixSuffixSaver
	stderr *prefixSuffixSaver
}

// ExpectSuccess the command did not execute successful
// (e.g. exit code != 0 on unix), a ExitCodeError is returned.
func (r *Result) ExpectSuccess() error {
	if !r.success {
		return &ExitCodeError{Result: r}
	}

	return nil
}

type ResultOut struct {
	*Result
	CombinedOutput []byte
}

func (r *ResultOut) StrOutput() string {
	return string(r.CombinedOutput)
}
