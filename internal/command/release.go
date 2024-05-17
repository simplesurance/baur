package command

import (
	"github.com/spf13/cobra"
)

var releaseCmd = &cobra.Command{
	Use:   "release",
	Short: "manage releases",
}

func init() {
	rootCmd.AddCommand(releaseCmd)
}
