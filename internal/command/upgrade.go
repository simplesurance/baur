package command

import (
	"github.com/spf13/cobra"
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "upgrade configuration files and the database schema",
}

func init() {
	rootCmd.AddCommand(upgradeCmd)
}
