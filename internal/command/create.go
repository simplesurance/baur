package command

import (
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "create releases",
}

func init() {
	rootCmd.AddCommand(createCmd)
}
