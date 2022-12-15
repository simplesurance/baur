package command

// TODO:
// - add help description
// - add parameter to specify shell / evaluate somehow which shell to use
// - add parameter to baur conf to specify default shell?
// - configure shell completion

import (
	"errors"
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur/v3/internal/command/term"
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
	if runtime.GOOS != "linux" {
		stderr.ErrPrintln(errors.New("This command is only supported on Linux."))
		os.Exit(1)
	}

	alwaysAllowed := []string{
		".baur.toml",
		".git/",
	}

	// TODO: bind mount read-only + print warning
	// TODO: print message stating that no files can be changed in the repo-dir

	repo := mustFindRepository()
	if len(repo.Cfg.TaskIsolation.ShellCommand) == 0 {
		exitWithErrf("shell command to execute is unknown.\nPlease specify the %s field in the %s configuration file.",
			term.Highlight("TaskIsolation.ShellCommand"),
			baur.RepositoryCfgFile)

	}

	vcsState := mustGetRepoState(repo.Path)
	task := mustArgToTask(repo, vcsState, args[0])
	inputs, err := baur.NewInputResolver(vcsState).Resolve(ctx, repo.Path, task)
	exitOnErr(err, "resolving task inputs failed")

	reExecInfoBuf, err := (&sandboxReExecInfo{
		RepositoryDir:       repo.Path,
		OverlayFsTmpDir:     overlayDir,
		Command:             []string{"bash"},
		AllowedFilesRelPath: append(relFileInputPaths(inputs), alwaysAllowed...),
	}).encode()

	exitOnErr(err, "encoding info for _sandbox_reexec failed")

	reExecArgs := []string{
		fmt.Sprintf("--verbose=%t", verboseFlag),
		"__sandbox_reexec",
	}
	err = sandbox.ReExecInNs(ctx, reExecArgs, reExecInfoBuf)
	exitOnErr(err)
}

func relFileInputPaths(inputs []baur.Input) []string {
	result := make([]string, 0, len(inputs))

	for _, input := range inputs {
		inputFile, ok := input.(*baur.InputFile)
		if !ok {
			continue
		}

		result = append(result, inputFile.RelPath())
	}

	return result
}
