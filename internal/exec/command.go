// Package exec runs external commands
package exec

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
)

var (
	// DefaultDebugfFn is the default debug print function.
	DefaultDebugfFn = func(string, ...interface{}) {}
	// DefaultDebugPrefix is the default prefix that is prepended to messages passed to the debugf function.
	DefaultDebugPrefix = "exec: "
)

// ExitCodeError is returned from Run() when a command exited with a code != 0.
type ExitCodeError struct {
	*Result
}

// Error returns the error description.
func (e ExitCodeError) Error() string {
	return fmt.Sprintf("exec: running '%s' in directory '%s' exited with code %d, expected 0, output: '%s'",
		e.Command, e.Dir, e.ExitCode, e.Output)
}

// Cmd represents a command that can be run.
type Cmd struct {
	path string
	args []string

	dir           string
	debugfFn      func(format string, v ...interface{})
	debugfPrefix  string
	expectSuccess bool
}

// Command returns a new Cmd struct.
// If name contains no path separators, Command uses LookPath to
// resolve name to a complete path if possible. Otherwise it uses name directly
// as Path.
// By default a command is run in the current working directory.
func Command(name string, arg ...string) *Cmd {
	return &Cmd{
		path:         name,
		args:         arg,
		dir:          ".",
		debugfFn:     DefaultDebugfFn,
		debugfPrefix: DefaultDebugPrefix,
	}
}

// Directory changes the directory in which the command is executed.
func (c *Cmd) Directory(dir string) *Cmd {
	c.dir = dir
	return c
}

// DebugfFunc sets the debug function for the command. It accepts a
// printf-style printf function and call it for every line that the command
// prints to STDOUT and STDERR when it's run.
func (c *Cmd) DebugfFunc(fn func(format string, arg ...interface{})) *Cmd {
	c.debugfFn = fn
	return c
}

// DebugfPrefix sets a prefix that is prepended to the message that is passed to the Debugf function.
func (c *Cmd) DebugfPrefix(prefix string) *Cmd {
	c.debugfPrefix = prefix
	return c
}

// ExpectSuccess if called, Run() will return an error if the command did not
// exit with code 0.
func (c *Cmd) ExpectSuccess() *Cmd {
	c.expectSuccess = true
	return c
}

func cmdString(cmd *exec.Cmd) string {
	// cmd.Args[0] contains the command name, cmd.Path the absolute command path,
	// omit cmd.Args[0] from the string
	if len(cmd.Args) > 1 {
		return fmt.Sprintf("%s %v", cmd.Path, strings.Join(cmd.Args[1:], " "))
	}

	return cmd.Path
}

// Result describes the result of a run Cmd.
type Result struct {
	Command  string
	Dir      string
	Output   []byte
	ExitCode int
}

// StrOutput returns Result.Output as string.
func (r *Result) StrOutput() string {
	return string(r.Output)
}

// ExpectSuccess if the ExitCode in Result is not 0, the function returns an
// ExitCodeError for the execution.
func (r *Result) ExpectSuccess() error {
	if r.ExitCode == 0 {
		return nil
	}

	return ExitCodeError{Result: r}
}

func exitCodeFromErr(err error) (int, error) {
	var ee *exec.ExitError
	var ok bool

	if ee, ok = err.(*exec.ExitError); !ok {
		return 0, err
	}

	if status, ok := ee.Sys().(syscall.WaitStatus); ok {
		return status.ExitStatus(), nil
	}

	return 0, err
}

// Run executes the command.
func (c *Cmd) Run() (*Result, error) {
	cmd := exec.Command(c.path, c.args...)
	cmd.Dir = c.dir

	outReader, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = cmd.Stdout

	c.debugfFn(c.debugfPrefix+"running '%s' in directory '%s'", cmdString(cmd), cmd.Dir)
	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	var outBuf bytes.Buffer
	firstline := true
	in := bufio.NewScanner(outReader)
	for in.Scan() {
		if firstline {
			firstline = false
		} else {
			outBuf.WriteRune('\n')
		}

		c.debugfFn(c.debugfPrefix + in.Text())

		outBuf.Write(in.Bytes())
	}

	if err := in.Err(); err != nil {
		_ = cmd.Wait()

		return nil, err
	}

	var exitCode int
	waitErr := cmd.Wait()
	if exitCode, err = exitCodeFromErr(waitErr); err != nil {
		return nil, err
	}

	c.debugfFn(c.debugfPrefix+"command terminated with exitCode: %d", exitCode)

	result := Result{
		Command:  cmdString(cmd),
		Dir:      cmd.Dir,
		ExitCode: exitCode,
		Output:   outBuf.Bytes(),
	}

	if c.expectSuccess && exitCode != 0 {
		return nil, ExitCodeError{Result: &result}
	}

	return &result, nil
}
