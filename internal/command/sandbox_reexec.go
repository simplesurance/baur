package command

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur/v3/internal/exec/sandbox"
	"github.com/simplesurance/baur/v3/internal/log"
)

func init() {
	rootCmd.AddCommand(&newSandboxReexecCmd().Command)
}

type sandboxReexecCmd struct {
	cobra.Command
}

func newSandboxReexecCmd() *sandboxReexecCmd {
	cmd := sandboxReexecCmd{
		Command: cobra.Command{
			Use:               "__sandbox_reexec",
			Short:             "internal command",
			Hidden:            true,
			ValidArgsFunction: cobra.NoFileCompletions,
			PersistentPreRun: func(_ *cobra.Command, _ []string) {
				log.StdLogger = log.New(verboseFlag, "reexec: ") // TODO: better prefix?
			},
		},
	}
	cmd.Run = cmd.run

	return &cmd
}

func (c *sandboxReexecCmd) run(cmd *cobra.Command, args []string) {
	fd := os.NewFile(sandbox.ReExecDataPipeFD, "reexec-data")
	info, err := sandboxReExecInfoDecode(fd)
	_ = fd.Close()
	exitOnErrf(err, "decoding data from fd %d into %T failed", sandbox.ReExecDataPipeFD, &info)

	log.Debugf("received the following data from parent process: +%v", info)

	// TODO: rename info.OverlayFsTmpDir??
	tmpdir := filepath.Join(info.OverlayFsTmpDir, fmt.Sprint(os.Getpid()))
	hd, err := sandbox.HideFiles(info.RepositoryDir, tmpdir, info.AllowedFilesRelPath)
	exitOnErr(err, "hiding files in repository directory failed")

	err = runShell(ctx, info.Command, info.RepositoryDir)
	if err != nil {
		// TODO print a nicer error message to distinguish starting the command failed and it exited with code != 0
		stderr.ErrPrintf(err, "running shell failed")
	}

	// TODO: run cleanup function also when terminating because of a signal

	closeErr := hd.Close()
	if closeErr != nil {
		stderr.ErrPrintf(closeErr, "cleanup failed")
	}

	if err != nil || closeErr != nil {
		exitFunc(1)
	}
}

func runShell(ctx context.Context, command []string, workingDir string) error {
	// TODO: use our exec package
	stdout.Println("starting shell...")
	cmd := exec.CommandContext(ctx, "bash")
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Dir = workingDir
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Pdeathsig:  syscall.SIGKILL,
		Foreground: true,
	}

	return cmd.Run()
}

type sandboxReExecInfo struct {
	RepositoryDir       string
	OverlayFsTmpDir     string
	Command             []string
	AllowedFilesRelPath []string
}

func (s *sandboxReExecInfo) encode() (*bytes.Buffer, error) {
	var buf bytes.Buffer

	err := gob.NewEncoder(&buf).Encode(s)
	if err != nil {
		return nil, err
	}

	return &buf, nil
}

func sandboxReExecInfoDecode(r io.ReadCloser) (*sandboxReExecInfo, error) {
	var result sandboxReExecInfo

	err := gob.NewDecoder(r).Decode(&result)
	if err != nil {
		return nil, err
	}

	return &result, err
}
