package command

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/simplesurance/baur/v3/internal/command/term"
	"github.com/simplesurance/baur/v3/internal/exec"
	"github.com/simplesurance/baur/v3/internal/log"
	"github.com/simplesurance/baur/v3/internal/testutils/logwriter"
)

var testdataDir string

func init() {
	wd, err := os.Getwd()
	if err != nil {
		panic(wd)
	}

	testdataDir = filepath.Join(wd, "testdata")
}

// interceptCmdOutput changes the stdout and stderr streams to that the
// commands write to the returned buffers, all output is additionally still
// logged via the test logger
func interceptCmdOutput(t *testing.T) (stdoutBuf, stderrBuf *bytes.Buffer) {
	var bufStdout bytes.Buffer
	var bufStderr bytes.Buffer

	oldStdout := stdout
	stdout = term.NewStream(logwriter.New(t, &bufStdout))
	oldStderr := stderr
	stderr = term.NewStream(logwriter.New(t, &bufStderr))

	t.Cleanup(func() {
		stdout = oldStdout
		stderr = oldStderr
	})

	return &bufStdout, &bufStderr
}

type exitInfo struct {
	Code int
}

func (e *exitInfo) String() string {
	return fmt.Sprintf("program terminated with exit code: %d", e.Code)
}

// initTest does the following:
// - changes the exitFunc to panic instead of calling os.Exit(),
// - changes stdout and stderr streams for the command to be redirect to the test logger
// - changes the exec debug function to the test logger,
func initTest(t *testing.T) {
	t.Helper()

	exitFunc = func(code int) {
		panic(&exitInfo{Code: code})
	}

	redirectOutputToLogger(t)
}

func redirectOutputToLogger(t *testing.T) {
	// TODO: when tests are run in parallel this will cause unexpected
	// results, global package vars are modified that would affect all
	// parallel running tests
	log.RedirectToTestingLog(t)

	oldExecDebugFfN := exec.DefaultLogFn
	exec.DefaultLogFn = t.Logf

	oldStdout := stdout
	stdout = term.NewStream(logwriter.New(t, io.Discard))
	oldStderr := stderr
	stderr = term.NewStream(logwriter.New(t, io.Discard))

	t.Cleanup(func() {
		exec.DefaultLogFn = oldExecDebugFfN
		stdout = oldStdout
		stderr = oldStderr
	})
}

// interceptExitCode changes the exitFunc to store the exit Code in
// resultExitCode.
// If the executed command does not exit with code 0, it will not panic.
// The previous exitFunc will be restored when the test finished.
func interceptExitCode(t *testing.T, resultExitCode *int) {
	oldExitFunc := exitFunc
	exitFunc = func(code int) {
		*resultExitCode = code
	}

	t.Cleanup(func() {
		exitFunc = oldExitFunc
	})
}
