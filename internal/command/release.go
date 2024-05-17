package command

import (
	"github.com/spf13/cobra"
)

var releaseCMd = &cobra.Command{
	Use:   "release",
	Short: "manage releases",
}

func init() {
	rootCmd.AddCommand(releaseCMd)
}
