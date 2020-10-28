package command

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/simplesurance/baur/v1/internal/command/term"
	"github.com/simplesurance/baur/v1/internal/exec"
	"github.com/simplesurance/baur/v1/internal/log"
	"github.com/simplesurance/baur/v1/internal/testutils/logwriter"
)

// interceptCmdOutput changes the stdout and stderr streams to that the
// commands write to the returned buffers, all output is additionally still
// logged via the test logger
func interceptCmdOutput(t *testing.T) (stdoutBuf, stderrBuf *bytes.Buffer) {
	var bufStdout bytes.Buffer
	var bufStderr bytes.Buffer

	stdout = term.NewStream(logwriter.New(t, &bufStdout))
	stderr = term.NewStream(logwriter.New(t, &bufStderr))

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

	redirectOutputToLogger(t)
}

func redirectOutputToLogger(t *testing.T) {
	log.StdLogger.SetOutput(log.NewTestLogOutput(t))
	exec.DefaultDebugfFn = t.Logf
	stdout = term.NewStream(logwriter.New(t, ioutil.Discard))
	stderr = term.NewStream(logwriter.New(t, ioutil.Discard))
}
