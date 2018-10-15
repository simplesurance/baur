package command

import (
	"github.com/spf13/cobra"
)

const repoLongHelp = `
The repo command groups subcommands that acts on the repository,
like initializing a repository config.
`

var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "init a repository config",
	Long:  repoLongHelp[1:],
}

func init() {
	rootCmd.AddCommand(repoCmd)
}
