package command

import (
	"github.com/spf13/cobra"
)

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "list inputs that differ between two task-runs",
}

func init() {
	rootCmd.AddCommand(diffCmd)
}
