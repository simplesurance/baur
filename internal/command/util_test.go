package command

import (
	"bytes"
	"testing"

	"github.com/simplesurance/baur/v1/internal/command/term"
	"github.com/simplesurance/baur/v1/internal/exec"
)

// interceptCmdOutput changes the stdout and stderr streams to that the
// commands write to the returned buffers
func interceptCmdOutput() (stdoutBuf, stderrBuf *bytes.Buffer) {
	var bufStdout bytes.Buffer
	var bufStderr bytes.Buffer

	stdout = term.NewStream(&bufStdout)
	stderr = term.NewStream(&bufStderr)

	return &bufStdout, &bufStderr
}

// initTest does the following:
// - changes the exitFunc to fail the testcase when it is called with an exitCode !=0.
// - changes the exec debug function to the test logger,
func initTest(t *testing.T) {
	t.Helper()

	exitFunc = func(code int) {
		t.Fatalf("baur command exited with code %d", code)
	}

	exec.DefaultDebugfFn = t.Logf
}
