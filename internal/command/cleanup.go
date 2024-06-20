package command

import "github.com/spf13/cobra"

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "delete old database records",
}

func init() {
	rootCmd.AddCommand(cleanupCmd)
}
