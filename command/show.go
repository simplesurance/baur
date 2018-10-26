package command

import (
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "show information about apps, builds, inputs",
}

func init() {
	rootCmd.AddCommand(showCmd)
}
