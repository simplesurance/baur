package command

// TODO:
// - add help description
// - add parameter to specify shell / evaluate somehow which shell to use
// - add parameter to baur conf to specify default shell?
// - configure shell completion

import (
	"bytes"
	"encoding/gob"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur/v3/internal/exec/sandbox"
	"github.com/simplesurance/baur/v3/pkg/baur"
)

// TODO: change this dir to a reasonable directory, make it configurable
const overlayDir = "/home/fho/tmp"

func init() {
	rootCmd.AddCommand(&newShellCmd().Command)
}

type shellCmd struct {
	cobra.Command
}

func newShellCmd() *shellCmd {
	cmd := shellCmd{
		Command: cobra.Command{
			Use:   "shell TASK-ID",
			Short: "Start a shell in the run environment of a task",
			ValidArgsFunction: newCompleteTargetFunc(completeTargetFuncOpts{
				withoutWildcards: true,
				withoutPaths:     true,
				withoutAppNames:  true,
			}),
			Args: cobra.ExactArgs(1),
		},
	}
	cmd.Run = cmd.run

	return &cmd
}

func (c *shellCmd) run(cmd *cobra.Command, args []string) {
	var data bytes.Buffer

	// TODO: bind mount read-only + print warning
	// TODO: print message stating that no files can be changed in the repo-dir

	repo := mustFindRepository()
	vcsState := mustGetRepoState(repo.Path)
	task := mustArgToTask(repo, vcsState, args[0])
	inputs, err := baur.NewInputResolver(vcsState).Resolve(ctx, repo.Path, task)
	exitOnErr(err, "resolving task inputs failed")
	for _, input := range inputs {
		inputFile, ok := input.(*baur.InputFile)
		if !ok {
			continue
		}
		inputFile.RelPath()

	}

	reexecData := sandboxReExecInfo{
		RepositoryDir:   repo.Path,
		OverlayFsTmpDir: overlayDir,
		Command:         []string{"bash"},
	}
	err := gob.NewEncoder(&data).Encode(&reexecData)
	exitOnErr(err, "encoding info for _sandbox_reexec failed: %w", err)

	reExecArgs := []string{
		fmt.Sprintf("--verbose=%t", verboseFlag),
		"__sandbox_reexec",
	}
	err = sandbox.ReExecInNs(ctx, reExecArgs, &data)
	exitOnErr(err)
}

type sandboxReExecInfo struct {
	RepositoryDir   string
	OverlayFsTmpDir string
	Command         []string
}
