package command

import (
	"github.com/spf13/cobra"
)

const buildsLongHelp = `
The build command groups subcommands that act on past application builds.
`

var buildsCmd = &cobra.Command{
	Use:   "builds",
	Short: "show informations about past builds",
	Long:  buildsLongHelp[1:],
}

func init() {
	rootCmd.AddCommand(buildsCmd)
}
