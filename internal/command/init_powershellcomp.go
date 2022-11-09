package command

import (
	"github.com/spf13/cobra"
)

type initPowerShellCompCmd struct {
	cobra.Command
}

func init() {
	initCmd.AddCommand(&newInitPowerShellCompCmd().Command)
}

const initPowerShellCompLongHelp = `
Generate a completion script for PowerShell.

The generated script is printed to stdout.
`

func newInitPowerShellCompCmd() *initPowerShellCompCmd {
	cmd := initPowerShellCompCmd{
		Command: cobra.Command{
			Use:   "powershellcomp",
			Short: "generate a powershell completion script",
			Long:  initPowerShellCompLongHelp,
		},
	}

	cmd.Run = cmd.run

	return &cmd
}

func (c *initPowerShellCompCmd) run(_ *cobra.Command, _ []string) {
	err := rootCmd.GenPowerShellCompletion(stdout)
	exitOnErr(err)
}
