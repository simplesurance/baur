//go:build linux || freebsd

package exec

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	stdlib_exec "os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	// test approach is based on go/src/syscall/syscall_linux_test.go
	switch {
	case os.Getenv("BAUR_EXEC_DEATHSIG_TEST_PARENT") == "1":
		deathSignalParent()

	case os.Getenv("BAUR_EXEC_DEATHSIG_TEST_CHILD") == "1":
		deathSignalChild()

	default:
		os.Exit(m.Run())
	}
}

func deathSignalParent() {
	DefaultDebugfFn = func(format string, a ...any) { fmt.Printf(format, a...) }

	cmd := Command(os.Args[0]).
		SetEnv([]string{
			"BAUR_EXEC_DEATHSIG_TEST_PARENT=",
			"BAUR_EXEC_DEATHSIG_TEST_CHILD=1",
		})

	_, err := cmd.Run()
	if err != nil {
		fmt.Printf("parent: executing child process failed: %s\n", err)
		os.Exit(2)
	}

	// the following  should not never be reached, because th process is
	// getting killed
	fmt.Println("parent: child process executed before parent, expecting parent to get killed first")
	os.Exit(4)
}

func deathSignalChild() {
	fmt.Printf("child started, pid: %d\n", os.Getpid())
	select {}
}

func TestProcessTerminatesWithParent(t *testing.T) {
	cmd := stdlib_exec.Command(os.Args[0])
	cmd.Env = append(os.Environ(), "BAUR_EXEC_DEATHSIG_TEST_PARENT=1")

	stdoutReader, err := cmd.StdoutPipe()
	assert.NoError(t, err, "opening stdout pipe for new cmd failed")
	cmd.Stderr = cmd.Stdout

	err = cmd.Start()
	assert.NoError(t, err, "starting parent process failed")

	t.Log("started parent process, waiting for it to print that it started the child")

	stdoutBuf := bufio.NewReader(stdoutReader)
	var parentStdoutLine string
	for {
		parentStdoutLine, err = stdoutBuf.ReadString('\n')
		if err != nil {
			t.Error(t, err, "reading from stdout of parent process failed")
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			t.FailNow()
		}

		t.Logf("read from parent's process stdout: %q", parentStdoutLine)
		parentStdoutLine = strings.TrimPrefix(parentStdoutLine, DefaultDebugPrefix)

		if strings.HasPrefix(parentStdoutLine, "running ") {
			continue
		}

		if strings.HasPrefix(parentStdoutLine, "child started") {
			break
		}

		t.Errorf("got unexpected stdout output: %q", parentStdoutLine)
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.FailNow()
	}

	childPidStr := strings.TrimSpace(strings.TrimPrefix(parentStdoutLine, "child started, pid: "))
	childPid, err := strconv.Atoi(childPidStr)
	if err != nil {
		t.Errorf("could not extract pid from process output: %q", parentStdoutLine)
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.FailNow()
	}
	t.Logf("child process pid: %d", childPid)

	t.Log("killing parent process")
	err = cmd.Process.Kill()
	assert.NoError(t, err, "killing parent process failed")

	time.Sleep(2 * time.Second)
	out, err := io.ReadAll(stdoutReader)
	if err != nil {
		t.Logf("could not read remaining output of parent process: %v", err)
	} else if len(out) > 0 {
		t.Logf("remaining parent process output: %q", string(out))
	}
	err = cmd.Wait()

	require.Error(t, err, "parent process execution succeeded, expected termination with exit code 137")

	var exitErr *stdlib_exec.ExitError
	if errors.As(err, &exitErr) {
		t.Logf("parent process terminated with exit code %d", exitErr.ExitCode())
		assert.False(t, exitErr.Exited(), "parent process did not terminate because of signal")
	} else {
		t.Fatalf("parent process execution failed with unexpected error: %s", err)
	}

	// on unix FindProcesss succeeds and returns a process also of the process terminated
	p, err := os.FindProcess(childPid)
	require.NoError(t, err, "finding child process failed")

	err = p.Signal(syscall.Signal(0))
	require.ErrorIs(t, err, os.ErrProcessDone, "child process is still running")
}
