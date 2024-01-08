package exec

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"syscall"
	"time"

	"github.com/fatih/color"
)

const defMaxErrOutputBytesPerStream = 16 * 1024

type PrintfFn func(format string, a ...any)

var (
	// DefaultLogFn is the default debug print function.
	DefaultLogFn PrintfFn
	// DefaultLogPrefix is the default prefix that is prepended to messages passed to the debugf function.
	DefaultLogPrefix = "exec: "
	// DefaultStderrColorFn is the default function that is used to colorize stderr output that is streamed to the log function.
	DefaultStderrColorFn = color.New(color.FgRed).SprintFunc()
)

type Cmd struct {
	name string
	args []string
	dir  string
	env  []string

	stdout io.Writer
	stderr io.Writer

	maxStoredErrBytesPerStream int

	expectSuccess bool

	logFn                    PrintfFn
	logPrefix                string
	logFnStderrStreamColorfn func(a ...any) string
}

// Command returns an executable representation of a Command.
func Command(name string, arg ...string) *Cmd {
	return &Cmd{
		name:                       name,
		args:                       arg,
		logPrefix:                  DefaultLogPrefix,
		logFn:                      DefaultLogFn,
		stdout:                     nil,
		stderr:                     nil,
		maxStoredErrBytesPerStream: defMaxErrOutputBytesPerStream,
		logFnStderrStreamColorfn:   DefaultStderrColorFn,
	}
}

// Stdout streams the standard output to w during execution.
func (c *Cmd) Stdout(w io.Writer) *Cmd {
	c.stdout = w
	return c
}

// Stdout streams the standard error output to w during execution.
func (c *Cmd) Stderr(w io.Writer) *Cmd {
	c.stderr = w
	return c
}

// LogFn sets a Printf-style function as logger.
func (c *Cmd) LogFn(fn PrintfFn) *Cmd {
	c.logFn = fn
	return c
}

// ExpectSuccess if called, Run() will return an error if the command did not
// exit with code 0.
func (c *Cmd) ExpectSuccess() *Cmd {
	c.expectSuccess = true
	return c
}

// Directory defines the directiory in which the command is executed.
func (c *Cmd) Directory(dir string) *Cmd {
	c.dir = dir
	return c
}

// LogPrefix defines a string that is used as prefix for all messages written via *Cmd.LogFn.
func (c *Cmd) LogPrefix(prefix string) *Cmd {
	c.logPrefix = prefix
	return c
}

// Env defines environment variables that are set during execution.
func (c *Cmd) Env(env []string) *Cmd {
	c.env = env
	return c
}

func (c *Cmd) logf(format string, a ...any) {
	if c.logFn == nil {
		return
	}
	c.logFn(c.logPrefix+format, a...)
}

func exitCodeFromErr(waitErr error) (exitCode int, err error) {
	var ee *exec.ExitError
	var ok bool

	if waitErr == nil {
		return 0, err
	}

	if ee, ok = waitErr.(*exec.ExitError); !ok {
		return -1, waitErr
	}

	if status, ok := ee.Sys().(syscall.WaitStatus); ok {
		return status.ExitStatus(), nil
	}

	return -1, waitErr
}

func newMultiWriter(w ...io.Writer) io.Writer {
	writers := make([]io.Writer, 0, len(w))
	for _, ww := range w {
		if ww != nil && ww != io.Discard {
			writers = append(writers, ww)
		}
	}
	return io.MultiWriter(writers...)
}

func (c *Cmd) startOutputStreamLogging(name string, useColorStderrColorfn bool) (io.Writer, func() error) {
	logReader, logWriter := io.Pipe()
	scannerTerminated := make(chan struct{})

	go func() {
		sc := bufio.NewScanner(logReader)
		// use a bigger buf to make it more unlikely that it will fail because of very long lines
		sc.Buffer([]byte{}, 512*1024)

		for sc.Scan() {
			if useColorStderrColorfn && c.logFnStderrStreamColorfn != nil {
				c.logf(c.logFnStderrStreamColorfn(sc.Text(), "\n"))
			} else {
				c.logf(sc.Text() + "\n")
			}
		}

		if err := sc.Err(); err != nil {
			_ = logReader.CloseWithError(fmt.Errorf("%s: streaming output failed: %w", name, err))
		}
		close(scannerTerminated)
	}()

	return logWriter, func() error {
		err := logWriter.Close()
		<-scannerTerminated

		if err != nil {
			return fmt.Errorf("%s: closing output stream failed: %w", name, err)
		}

		return nil
	}
}

// Run executes the command.
func (c *Cmd) Run(ctx context.Context) (*Result, error) {
	cmd := exec.CommandContext(ctx, c.name, c.args...)
	cmd.SysProcAttr = defSysProcAttr()
	cmd.WaitDelay = time.Minute
	cmd.Dir = c.dir
	cmd.Env = c.env

	stdoutLogWriterCloseFn := func() error { return nil }
	stderrLogWriterCloseFn := func() error { return nil }
	var stdoutLogWriter, stderrLogWriter io.Writer
	if c.logFn != nil {
		stdoutLogWriter, stdoutLogWriterCloseFn = c.startOutputStreamLogging("stdout", false)
		stderrLogWriter, stderrLogWriterCloseFn = c.startOutputStreamLogging("stderr", true)
	}

	// if a write to one of the writers of the MultiWriter fails, the command will fail
	stdoutPss := prefixSuffixSaver{N: c.maxStoredErrBytesPerStream}
	cmd.Stdout = newMultiWriter(c.stdout, stdoutLogWriter, &stdoutPss)

	stderrPss := prefixSuffixSaver{N: c.maxStoredErrBytesPerStream}
	cmd.Stderr = newMultiWriter(c.stderr, stderrLogWriter, &stderrPss)

	// lock to thread because of:
	// https://github.com/golang/go/issues/27505#issuecomment-713706104
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	c.logf("running %q in directory %q\n", cmd.String(), cmd.Dir)

	err := cmd.Start()
	if err != nil {
		return nil, errors.Join(err, stdoutLogWriterCloseFn(), stderrLogWriterCloseFn())
	}

	waitErr := cmd.Wait()
	logWriterErr := errors.Join(stdoutLogWriterCloseFn(), stderrLogWriterCloseFn())

	if waitErr != nil && ctx.Err() != nil {
		return nil, errors.Join(ctx.Err(), waitErr, logWriterErr)
	}

	if logWriterErr != nil {
		return nil, errors.Join(logWriterErr, waitErr)
	}

	exitCode, exitCodeErr := exitCodeFromErr(waitErr)
	if exitCodeErr != nil {
		return nil, exitCodeErr
	}

	c.logf("command terminated with exitCode: %d\n", exitCode)

	result := Result{
		Command:  cmd.String(),
		Dir:      cmd.Dir,
		ExitCode: exitCode,
		stdout:   &stdoutPss,
		stderr:   &stderrPss,
	}
	if c.expectSuccess && exitCode != 0 {
		return nil, ExitCodeError{Result: &result}
	}

	return &result, nil
}

// RunCombinedOut executes the command and stores the combined stdout and
// stderr output of the process in ResultOut.CombinedOutput.
func (c *Cmd) RunCombinedOut(ctx context.Context) (*ResultOut, error) {
	buf := bytes.Buffer{}

	if c.stdout != nil {
		return nil, errors.New("Cmd.stdout must be nil")
	}
	if c.stderr != nil {
		return nil, errors.New("Cmd.stderr must be nil")
	}

	c.stdout = &buf
	c.stderr = c.stdout

	result, err := c.Run(ctx)
	if err != nil {
		return nil, err
	}

	return &ResultOut{Result: result, CombinedOutput: buf.Bytes()}, nil
}
