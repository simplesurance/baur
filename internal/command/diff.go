package command

import (
	"github.com/spf13/cobra"
)

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "list inputs that differ between two builds",
}

func init() {
	rootCmd.AddCommand(diffCmd)
}
