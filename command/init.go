package command

import (
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "initialize configuration files and the baur database",
}

func init() {
	rootCmd.AddCommand(initCmd)
}
