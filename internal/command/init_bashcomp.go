package command

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur/v1/internal/command/term"
	"github.com/simplesurance/baur/v1/internal/fs"
	"github.com/simplesurance/baur/v1/internal/log"
)

const bashCompLongHelp = `
Installs a bash completion file for baur.
The bash_completion file is written into the user's bash completion directory.

Environment Variables:
BASH_COMPLETION_USER_DIR	Destination directory
`

// bashCompCmd represents the completion command
var bashCompCmd = &cobra.Command{
	Use:   "bashcomp",
	Short: "generate and install a bash completion file for baur",
	Long:  strings.TrimSpace(bashCompLongHelp),
	Run:   bashComp,
}

func init() {
	initCmd.AddCommand(bashCompCmd)
}

func getBashCompletionDir() string {
	var exist bool

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

	if path, exist := os.LookupEnv("BASH_COMPLETION_USER_DIR"); exist {
		return path
	}

	var xdgHome string
	if xdgHome, exist = os.LookupEnv("XDG_DATA_HOME"); exist {
		return filepath.Join(xdgHome, "bash-completion/completions")
	}

	if home, exist := os.LookupEnv("HOME"); exist {
		return filepath.Join(home, ".local/share/bash-completion/completions")
	}

	return "~/.local/share/bash-completion/completions"
}

func mustCreatebashComplDir(path string) {
	isDir, err := fs.IsDir(path)
	if err == nil {
		if isDir {
			return
		}

		if !isDir {
			log.Fatalf("'%s' must be a directory", path)
		}
	}

	if !os.IsNotExist(err) {
		log.Fatalln(err)
	}

	err = fs.Mkdir(path)
	exitOnErrf(err, "could not create bash completion dir %q", path)
}

func bashComp(cmd *cobra.Command, args []string) {
	complDir := getBashCompletionDir()

	mustCreatebashComplDir(complDir)

	complFile := filepath.Join(complDir, "baur")
	f, err := os.Create(complFile)
	exitOnErrf(err, "creating %q' failed", complFile)

	err = rootCmd.GenBashCompletion(f)
	exitOnErr(err, "generating bash completion failed")

	err = f.Close()
	exitOnErrf(err, "closing %q failed", complFile)

	stdout.Printf("bash completion file written to %s\n", term.Highlight(complFile))
}
