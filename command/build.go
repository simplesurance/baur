package command

import (
	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "app builds related commands",
}

func init() {
	rootCmd.AddCommand(buildCmd)
}
