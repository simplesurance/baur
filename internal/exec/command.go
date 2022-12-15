// Package exec runs external commands
package exec

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/simplesurance/baur/v3/internal/log"
)

var (
	// DefaultDebugfFn is the default debug print function.
	DefaultDebugfFn = func(string, ...interface{}) {}
	// DefaultDebugPrefix is the default prefix that is prepended to messages passed to the debugf function.
	DefaultDebugPrefix = "exec: "
)

// Cmd represents a command that can be run.
type Cmd struct {
	*exec.Cmd

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
	cmd := exec.Command(name, arg...)
	cmd.SysProcAttr = defSysProcAttr()

	return &Cmd{
		Cmd:          cmd,
		debugfFn:     DefaultDebugfFn,
		debugfPrefix: DefaultDebugPrefix,
	}
}

// Directory changes the directory in which the command is executed.
func (c *Cmd) Directory(dir string) *Cmd {
	c.Cmd.Dir = dir
	return c
}

// SetEnv sets the environment variables that the process uses.
// Each element is in the format KEY=VALUE.
func (c *Cmd) SetEnv(env []string) *Cmd {
	c.Cmd.Env = env
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

// Run executes the command.
func (c *Cmd) Run() (*Result, error) {
	cmd := c.Cmd

	outReader, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = cmd.Stdout

	// lock to thread because of:
	// https://github.com/golang/go/issues/27505#issuecomment-713706104
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	c.debugfFn(c.debugfPrefix+"running '%s' in directory '%s'\n", cmdString(cmd), cmd.Dir)
	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	var outBuf bytes.Buffer
	firstline := true
	in := bufio.NewScanner(outReader)
	for in.Scan() {
		c.debugfFn(c.debugfPrefix + in.Text() + "\n")
		if firstline {
			firstline = false
		} else {
			outBuf.WriteRune('\n')
		}

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

	c.debugfFn(c.debugfPrefix+"command terminated with exitCode: %d\n", exitCode)

	result := Result{
		Command:  cmdString(cmd),
		Dir:      cmd.Dir,
		ExitCode: exitCode,
		Output:   outBuf.Bytes(),
	}
	if result.Dir == "" {
		result.Dir = "."
	}

	if c.expectSuccess && exitCode != 0 {
		return nil, ExitCodeError{Result: &result}
	}

	return &result, nil
}

const ReExecDataPipeFD uintptr = 3

// ReExecInNs executes the current running binary (/proc/self/exe) again in
// a new Linux user- and mount Namespace.
// args are passed as command line arguments.
// data is piped to the the process via file-descriptor 3. If data is passed,
// the process must read it, otherwise the operation will fail.
// data is usually information that tells the new process what to do.
// The reexecuted process will run with uid & gid 0 but have the same
// permissions then the currently executing process.
func (c *Cmd) RunInNs(data io.Reader) (*Result, error) {
	//cmd := exec.CommandContext(ctx, "/proc/self/exe", args...)
	cmd := c.Cmd

	outReader, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = os.Stdout
	cmd.Stdin = os.Stdin

	xtraReader, xtraWriter, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	defer xtraReader.Close()
	// xtraWriter is closed without defer
	cmd.ExtraFiles = []*os.File{xtraReader}

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Unshareflags: syscall.CLONE_NEWUSER | syscall.CLONE_NEWNS,
		UidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      os.Getuid(),
				Size:        1,
			},
		},
		GidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      os.Getgid(),
				Size:        1,
			},
		},
	}

	log.Debugf("starting new process of myself (%q) in new user and mount namespaces", cmd)

	// lock to thread because of:
	// https://github.com/golang/go/issues/27505#issuecomment-713706104
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	err = cmd.Start()
	if err != nil {
		xtraWriter.Close()
		return nil, err
	}

	err = xtraWriter.SetWriteDeadline(time.Now().Add(30 * time.Second))
	if err != nil {
		log.Debugf("WARN: setting deadling on pipe failed: %s", err)
	}

	_, err = io.Copy(xtraWriter, data)
	if err != nil {
		err = fmt.Errorf("piping data to child process failed: %w", err)
		killErr := cmd.Process.Kill()
		if killErr != nil {
			return nil, fmt.Errorf("%s, killing the child process (pid: %d) too: %s",
				err, cmd.Process.Pid, killErr)
		}

		return nil, err
	}
	_ = xtraWriter.Close()

	var outBuf bytes.Buffer
	firstline := true
	in := bufio.NewScanner(outReader)
	for in.Scan() {
		c.debugfFn(c.debugfPrefix + in.Text() + "\n")
		if firstline {
			firstline = false
		} else {
			outBuf.WriteRune('\n')
		}

		outBuf.Write(in.Bytes())
	}

	if err := in.Err(); err != nil {
		err := fmt.Errorf("reading from stdout pipe failed: %w", err)
		if waitErr := cmd.Wait(); waitErr != nil {
			return nil, fmt.Errorf("%s, executing the command failed too: %s", err, waitErr)
		}

		return nil, err
	}

	var exitCode int
	waitErr := cmd.Wait()
	if exitCode, err = exitCodeFromErr(waitErr); err != nil {
		return nil, err
	}

	c.debugfFn(c.debugfPrefix+"command terminated with exitCode: %d\n", exitCode)

	result := Result{
		Command:  cmdString(cmd),
		Dir:      cmd.Dir,
		ExitCode: exitCode,
		Output:   outBuf.Bytes(),
	}
	if result.Dir == "" {
		result.Dir = "."
	}

	if c.expectSuccess && exitCode != 0 {
		return nil, ExitCodeError{Result: &result}
	}

	return &result, nil

}

func cmdString(cmd *exec.Cmd) string {
	// cmd.Args[0] contains the command name, cmd.Path the absolute command path,
	// omit cmd.Args[0] from the string
	if len(cmd.Args) > 1 {
		return fmt.Sprintf("%s %v", cmd.Path, strings.Join(cmd.Args[1:], " "))
	}

	return cmd.Path
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
