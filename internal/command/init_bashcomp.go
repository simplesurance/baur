package command

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur/v3/internal/command/term"
	"github.com/simplesurance/baur/v3/internal/fs"
)

const bashCompLongHelp = `
Installs a bash completion file for baur.
The bash_completion file is written into the user's bash completion directory.

Environment Variables:
BASH_COMPLETION_USER_DIR	Destination directory
`

type initBashCompCmd struct {
	cobra.Command
	stdout bool
}

func newInitBashCompCmd() *initBashCompCmd {
	cmd := initBashCompCmd{
		Command: cobra.Command{
			Use:   "bashcomp",
			Short: "generate and install a bash completion file for baur",
			Long:  strings.TrimSpace(bashCompLongHelp),
		},
	}

	cmd.Run = cmd.run

	cmd.Flags().BoolVar(&cmd.stdout, "stdout", false, "write completion script to stdout")

	return &cmd
}

func init() {
	initCmd.AddCommand(&newInitBashCompCmd().Command)
}

func xdgDataHome() (string, error) {
	const envVar = "XDG_DATA_HOME"
	/*
		https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html
		$XDG_DATA_HOME defines the base directory relative to which user-specific
		data files should be stored. If $XDG_DATA_HOME is either not set or empty, a
		default equal to $HOME/.local/share should be used.
	*/
	if path := os.Getenv(envVar); path != "" {
		return path, nil
	}

	if runtime.GOOS == "windows" {
		return "", fmt.Errorf("%s environment variable is not set", envVar)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("finding home directory failed: %w", err)
	}

	return filepath.Join(home, ".local", "share"), nil
}

func getBashCompletionDir() (string, error) {
	const envVar = "BASH_COMPLETION_USER_DIR"
	/*
		 https://github.com/scop/bash-completion/blob/master/README.md

		Q. Where should I install my own local completions?

		A. Put them in the completions subdir of $BASH_COMPLETION_USER_DIR (defaults to
		$XDG_DATA_HOME/bash-completion or ~/.local/share/bash-completion if
		$XDG_DATA_HOME is not set) to have them loaded on demand. See also the next
		question's answer for considerations for these files' names, they apply here as
		well. Alternatively, you can write them directly in ~/.bash_completion which is
		loaded eagerly by our main script.
	*/

	if path := os.Getenv(envVar); path != "" {
		return path, nil
	}

	xdgHome, err := xdgDataHome()
	if err != nil {
		return "", fmt.Errorf("%s environment variable is empty and locating XDG_DATA_HOME failed: %w", envVar, err)
	}

	return filepath.Join(xdgHome, "bash-completion", "completions"), nil
}

func (c *initBashCompCmd) run(_ *cobra.Command, _ []string) {
	if c.stdout {
		err := rootCmd.GenBashCompletionV2(stdout, false)
		exitOnErr(err, "generating bash completion failed")
		return
	}

	complDir, err := getBashCompletionDir()
	exitOnErr(err,
		"could not find bash completion directory,",
		"try rerunning the command with '--stdout'",
	)

	err = fs.Mkdir(complDir)
	exitOnErrf(err, "could not create directory %q", complDir)

	complFile := filepath.Join(complDir, "baur")

	err = rootCmd.GenBashCompletionFileV2(complFile, false)
	exitOnErr(err, "generating bash completion failed")

	stdout.Printf("bash completion file written to %s\n", term.Highlight(complFile))
}
