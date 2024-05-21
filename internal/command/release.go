package command

import (
	"github.com/spf13/cobra"
)

var releaseCmd = &cobra.Command{
	Use:   "release",
	Short: "create and show releases",
}

func init() {
	rootCmd.AddCommand(releaseCmd)
}
