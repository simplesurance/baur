package command

import (
	"github.com/spf13/cobra"
)

const appsLongHelp = `
The apps command groups subcommands that act on applications and their
configurations in the repository.
`

var appsCmd = &cobra.Command{
	Use:   "apps",
	Short: "build and list applications, init configuration files",
	Long:  appsLongHelp[1:],
}

func init() {
	rootCmd.AddCommand(appsCmd)
}
