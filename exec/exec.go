package exec

import (
	"bufio"
	"os/exec"
	"strings"
	"syscall"
)

var debugOutputFn = func(string, ...interface{}) { return }

// SetDebugOutputFn configures the package to pass debug output to this function
func SetDebugOutputFn(fn func(format string, v ...interface{})) {
	debugOutputFn = fn
}

// Command runs the passed command in a shell in the passed dir.
// If the command exits with a code != 0, err is nil
func Command(dir, command string) (output string, exitCode int, err error) {
	cmd := exec.Command("sh", "-c", command)
	debugOutputFn("exec: running cmd '%s %s' in directory '%s'\n", cmd.Path, strings.Join(cmd.Args, " "), dir)

	outReader, err := cmd.StdoutPipe()
	if err != nil {
		return
	}

	cmd.Dir = dir
	cmd.Stderr = cmd.Stdout

	err = cmd.Start()
	if err != nil {
		return
	}

	in := bufio.NewScanner(outReader)
	for in.Scan() {
		o := in.Text()
		debugOutputFn("exec: cmd output: '%s'\n", o)
		output += o + "\n"
	}

	err = cmd.Wait()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			if status, ok := ee.Sys().(syscall.WaitStatus); ok {
				exitCode = status.ExitStatus()
				err = nil
			}
		}

		return
	}

	return
}
