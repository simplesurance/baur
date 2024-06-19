package command

import (
	"github.com/spf13/cobra"
)

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "list apps, runs, inputs",
}

func init() {
	rootCmd.AddCommand(lsCmd)
}
