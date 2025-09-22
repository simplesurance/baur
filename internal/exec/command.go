package exec

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/fatih/color"
)

const (
	maxErrOutputBytesPerStream   = 16 * 1024
	outputStreamLineReaderBufSiz = 512 * 1024
)

type PrintfFn func(format string, a ...any)

var (
	// DefaultLogFn is the default debug print function.
	DefaultLogFn PrintfFn
	// DefaultLogPrefix is the default prefix that is prepended to messages passed to the debugf function.
	DefaultLogPrefix = "exec: "
	// DefaultStderrColorFn is the default function that is used to colorize stderr output that is streamed to the log function.
	DefaultStderrColorFn = color.New(color.FgRed).SprintFunc()
)

// Cmd represent a command that can be executed as new process.
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

// Command creates a Cmd that executes the binary named name with the arguments args.
func Command(name string, arg ...string) *Cmd {
	return &Cmd{
		name:                       name,
		args:                       arg,
		logPrefix:                  DefaultLogPrefix,
		logFn:                      DefaultLogFn,
		stdout:                     nil,
		stderr:                     nil,
		maxStoredErrBytesPerStream: maxErrOutputBytesPerStream,
		logFnStderrStreamColorfn:   DefaultStderrColorFn,
	}
}

// Stdout streams the standard output to w during execution.
func (c *Cmd) Stdout(w io.Writer) *Cmd {
	c.stdout = w
	return c
}

// Stderr streams the standard error output to w during execution.
func (c *Cmd) Stderr(w io.Writer) *Cmd {
	c.stderr = w
	return c
}

// LogFn sets a Printf-style function as logger.
func (c *Cmd) LogFn(fn PrintfFn) *Cmd {
	c.logFn = fn
	return c
}

// ExpectSuccess when the command is executed and the execution of the process
// fails (e.g. exit status != 0 on unix) return an ExitCodeError instead of
// nil.
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
	r, w := io.Pipe()
	done := make(chan struct{})

	go func() {
		sc := bufio.NewScanner(r)
		// use a bigger buf to make it more unlikely that it will fail because of very long lines
		sc.Buffer([]byte{}, outputStreamLineReaderBufSiz)

		for sc.Scan() {
			if useColorStderrColorfn && c.logFnStderrStreamColorfn != nil {
				c.logf("%s\n", c.logFnStderrStreamColorfn(sc.Text()))
			} else {
				c.logf("%s\n", sc.Text())
			}
		}

		if err := sc.Err(); err != nil {
			if errors.Is(err, bufio.ErrTooLong) {
				c.logf("WARN: streaming %s output failed, shown output might be complete, lines are too long, requiring newlines after latest %dBytes in output\n",
					name, outputStreamLineReaderBufSiz)
			} else {
				c.logf("WARN: streaming %s output failed, shown output might be complete: %s\n", name, err)
			}
			// We do not Close the logReader with an error because
			// it would cause the MultiWriter to fail on all subsequent Write() operations for all streams,
			// the command could fail because it could not write to
			// stderr/sdout anymore.

			// drain the logReader until the end, to prevent blocks of the MultiWriter
			_, cpErr := io.Copy(io.Discard, r)
			if cpErr != nil {
				r.CloseWithError(fmt.Errorf("draining %s stream reader after line scanning failed, also failed: %w", name, errors.Join(err, cpErr)))
			}

		}
		close(done)
	}()

	return w, func() error {
		err := w.Close()
		<-done

		if err != nil {
			return fmt.Errorf("closing %s output stream failed: %w", name, err)
		}

		return nil
	}
}

func (c *Cmd) resolveDir(d string) string {
	if d != "" {
		return d
	}
	d, err := os.Getwd()
	if err != nil {
		c.logFn("WARN: determining current working directory failed: %s", err)
		return ""
	}
	return d
}

// Run executes the command.
// If the command could not be started an error is returned.
// If the command was started successfully, and terminated unsuccessfully no
// error is returned, except if *Cmd.ExpectSuccess() was called before.
func (c *Cmd) Run(ctx context.Context) (*Result, error) {
	cmd := exec.CommandContext(ctx, c.name, c.args...)
	cmd.SysProcAttr = defSysProcAttr()
	cmd.WaitDelay = time.Minute
	cmd.Dir = c.resolveDir(c.dir)
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
	cmd.Stdout = newMultiWriter(&stdoutPss, c.stdout, stdoutLogWriter)

	stderrPss := prefixSuffixSaver{N: c.maxStoredErrBytesPerStream}
	cmd.Stderr = newMultiWriter(&stderrPss, c.stderr, stderrLogWriter)

	// lock to thread because of:
	// https://github.com/golang/go/issues/27505#issuecomment-713706104
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	c.logf("running %q in directory %q\n", cmd.String(), cmd.Dir)

	err := cmd.Start()
	if err != nil {
		return nil, errors.Join(err, stdoutLogWriterCloseFn(), stderrLogWriterCloseFn())
	}

	err = cmd.Wait()
	logWriterErr := errors.Join(stdoutLogWriterCloseFn(), stderrLogWriterCloseFn())
	if err != nil && ctx.Err() != nil {
		return nil, errors.Join(ctx.Err(), err, logWriterErr)
	}

	if logWriterErr != nil {
		return nil, errors.Join(logWriterErr, err)
	}

	var ee *exec.ExitError
	if err != nil && !errors.As(err, &ee) {
		return nil, err
	}

	result := Result{
		Command:  cmd.String(),
		Dir:      cmd.Dir,
		ExitCode: cmd.ProcessState.ExitCode(),
		Success:  cmd.ProcessState.Success(),
		stdout:   &stdoutPss,
		stderr:   &stderrPss,
		ee:       ee,
	}
	c.logf("command terminated with exit code: %d\n", result.ExitCode)

	if c.expectSuccess && !result.Success {
		return nil, &ExitCodeError{Result: &result}
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
