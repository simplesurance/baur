package exec

import (
	"fmt"
	"os/exec"
	"strings"
)

// ExitCodeError is returned from Run() when a command exited with a code != 0.
type ExitCodeError struct {
	*Result
}

type SprintFn func(...any) string

func (e *ExitCodeError) ColoredError(highlightFn SprintFn, errorFn SprintFn, withCmdOutput bool) string {
	var result strings.Builder
	var stdoutExists bool

	result.WriteString("executing ")
	result.WriteString(e.Command)
	result.WriteString(" in \"")
	result.WriteString(e.Dir)
	result.WriteString("\" ")
	result.WriteString(errorFn("failed"))
	result.WriteString(": ")
	result.WriteString(e.ee.String())

	if !withCmdOutput || (len(e.stdout.Bytes()) == 0 && len(e.stderr.Bytes()) == 0) {
		return result.String()
	}

	result.WriteRune('\n')

	if b := e.stdout.Bytes(); len(b) > 0 {
		result.WriteString("### ")
		result.WriteString(highlightFn("stdout "))
		result.WriteString("###\n")
		result.WriteString(strings.TrimSpace(string(b)))
		result.WriteRune('\n')
		stdoutExists = true
	}

	if b := e.stderr.Bytes(); len(b) > 0 {
		if stdoutExists {
			result.WriteRune('\n')
		}
		result.WriteString("### ")
		result.WriteString(highlightFn("stderr "))
		result.WriteString("###\n")
		result.WriteString(errorFn(strings.TrimSpace(string(b))))
		result.WriteRune('\n')
	}

	return result.String()
}

// Error returns the error description.
func (e *ExitCodeError) Error() string {
	return e.ColoredError(fmt.Sprint, fmt.Sprint, true)

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
