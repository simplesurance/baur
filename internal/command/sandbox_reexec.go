package command

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"

	baur_exec "github.com/simplesurance/baur/v3/internal/exec"
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
	info, err := baur_exec.ReExecInfoDecode(fd)
	_ = fd.Close()
	exitOnErrf(err, "decoding data from fd %d into %T failed", sandbox.ReExecDataPipeFD, &info)

	log.Debugf("received the following data from parent process: +%v", info)

	// TODO: rename info.OverlayFsTmpDir??
	tmpdir := filepath.Join(info.OverlayFsTmpDir, fmt.Sprint(os.Getpid()))
	hd, err := sandbox.HideFiles(info.RepositoryDir, tmpdir, info.AllowedFilesRelPath)
	exitOnErr(err, "hiding files in repository directory failed")

	err = runShell(ctx, info.Command, info.WorkingDirectory)
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
	if len(command) == 0 {
		return errors.New("command to execute is unspecified")
	}
	var args []string

	// TODO: use our exec package
	name := command[0]
	if len(command) > 1 {
		args = command[1:]
	}

	stdout.Printf("starting %s...\n", name)
	cmd := exec.CommandContext(ctx, name, args...)
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
