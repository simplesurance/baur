package command

import "github.com/spf13/cobra"

var appCmd = &cobra.Command{
	Use:   "app",
	Short: "app related commands",
}

func init() {
	rootCmd.AddCommand(appCmd)
}
