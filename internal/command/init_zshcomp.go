package command

import (
	"github.com/spf13/cobra"
)

type initZshCompCmd struct {
	cobra.Command
}

func init() {
	initCmd.AddCommand(&newInitZshCompCmd().Command)
}

const initZshCompLongHelp = `
Generate a completion script for the zsh shell.

The generated script is printed to stdout.
`

func newInitZshCompCmd() *initZshCompCmd {
	cmd := initZshCompCmd{
		Command: cobra.Command{
			Use:     "zshcomp",
			Short:   "generate a zsh completion script",
			Long:    initZshCompLongHelp,
			GroupID: initShellCompletionGroupID,
		},
	}

	cmd.Run = cmd.run

	return &cmd
}

func (c *initZshCompCmd) run(_ *cobra.Command, _ []string) {
	err := rootCmd.GenZshCompletion(stdout)
	exitOnErr(err)
}
