package command

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur/v4/internal/command/term"
	"github.com/simplesurance/baur/v4/internal/fs"
)

const (
	envVarBashCompletionUserDir = "BASH_COMPLETION_USER_DIR"
	envVarXDGDataHome           = "XDG_DATA_HOME"
)

var initBashCompLongHelp = fmt.Sprintf(`
Generate and install a baur completion script for the bash shell.

If --stdout is passed, the completion script is written to STDOUT.
Otherwise it is written to the completion directory determined by the following
environment variables.

Environment Variables:
 %s	Directory the completion script is written to
 %s			If %s is empty, the script is written
				to %s.
`,
	envVarBashCompletionUserDir,
	envVarXDGDataHome,
	envVarBashCompletionUserDir,
	filepath.Join("$"+envVarXDGDataHome, bashCompRelDataHomeCompletionDir()),
)

type initBashCompCmd struct {
	cobra.Command
	stdout bool
}

func newInitBashCompCmd() *initBashCompCmd {
	cmd := initBashCompCmd{
		Command: cobra.Command{
			Use:               "bashcomp",
			Short:             "generate and install a bash completion script",
			Long:              strings.TrimSpace(initBashCompLongHelp),
			GroupID:           initShellCompletionGroupID,
			ValidArgsFunction: cobra.NoFileCompletions,
		},
	}

	cmd.Run = cmd.run

	cmd.Flags().BoolVar(&cmd.stdout, "stdout", false,
		"write completion script to stdout")

	return &cmd
}

func init() {
	initCmd.AddCommand(&newInitBashCompCmd().Command)
}

func xdgDataHome() (string, error) {
	/*
		https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html
		$XDG_DATA_HOME defines the base directory relative to which user-specific
		data files should be stored. If $XDG_DATA_HOME is either not set or empty, a
		default equal to $HOME/.local/share should be used.
	*/
	if path := os.Getenv(envVarXDGDataHome); path != "" {
		return path, nil
	}

	if runtime.GOOS == "windows" {
		return "", fmt.Errorf("%s environment variable is not set", envVarXDGDataHome)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("finding home directory failed: %w", err)
	}

	return filepath.Join(home, ".local", "share"), nil
}

func bashCompRelDataHomeCompletionDir() string {
	return filepath.Join("bash-completion", "completions")
}

func bashCompDir() (string, error) {
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

	if path := os.Getenv(envVarBashCompletionUserDir); path != "" {
		return path, nil
	}

	xdgHome, err := xdgDataHome()
	if err != nil {
		return "", fmt.Errorf("%s environment variable is empty and locating XDG_DATA_HOME failed: %w",
			envVarBashCompletionUserDir, err,
		)
	}

	return filepath.Join(xdgHome, bashCompRelDataHomeCompletionDir()), nil
}

func (c *initBashCompCmd) run(_ *cobra.Command, _ []string) {
	if c.stdout {
		err := rootCmd.GenBashCompletionV2(stdout, false)
		exitOnErr(err, "generating bash completion failed")
		return
	}

	complDir, err := bashCompDir()
	exitOnErr(err,
		"could not find bash completion directory,",
		"try rerunning the command with '--stdout'",
	)

	err = fs.Mkdir(complDir)
	exitOnErrf(err, "could not create directory %q", complDir)

	complFile := filepath.Join(complDir, "baur")

	err = rootCmd.GenBashCompletionFileV2(complFile, false)
	exitOnErr(err, "generating completion script failed")

	stdout.Printf(
		"bash completion script was written to %s.\n"+
			"To load completions in your current shell session run:\n"+
			"\t%s\n",
		term.Highlight(complFile),
		term.Highlight(fmt.Sprintf("source %s", complFile)),
	)
}
