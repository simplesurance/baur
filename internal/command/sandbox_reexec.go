package command

import (
	"context"
	"encoding/gob"
	"fmt"
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
			Use:   "__sandbox_reexec",
			Short: "internal command",
		},
	}
	cmd.Run = cmd.run

	return &cmd
}

func (c *sandboxReexecCmd) run(cmd *cobra.Command, args []string) {
	var info sandboxReExecInfo
	fd := os.NewFile(sandbox.ReExecDataPipeFD, "reexec-data")
	err := gob.NewDecoder(fd).Decode(&info)
	exitOnErrf(err, "decoding data from fd %d into %T failed", sandbox.ReExecDataPipeFD, &info)
	_ = fd.Close()

	// TODO: add prefix to all log messages, that indicates that messages are logged from child process
	log.Debugf("received the following data from parent process: +%v", info)

	// TODO: rename info.OverlayFsTmpDir??
	tmpdir := filepath.Join(info.OverlayFsTmpDir, fmt.Sprint(os.Getpid()))
	hd, err := sandbox.HideFiles(info.RepositoryDir, tmpdir, []string{".baur.toml"})
	exitOnErr(err, "hiding files in repository directory failed")

	err = runShell(ctx)
	if err != nil {
		stderr.ErrPrintf(err, "running shell failed")
	}

	closeErr := hd.Close()
	if closeErr != nil {
		stderr.ErrPrintf(err, "clean up failed")
	}

	if err != nil || closeErr != nil {
		exitFunc(1)
	}
}

func runShell(ctx context.Context) error {
	stdout.Println("starting shell...")
	cmd := exec.CommandContext(ctx, "bash")
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Pdeathsig:  syscall.SIGKILL,
		Foreground: true,
	}

	return cmd.Run()
}
