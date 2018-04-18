package cmd

import (
	"github.com/simplesurance/sisubuild/sblog"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(buildCmd)
}

var buildCmd = &cobra.Command{
	Use:   "build <PATH>|<APP-NAME>|all",
	Short: "builds an application",
	Run:   build,
	Args:  cobra.ExactArgs(1),
}

func build(cmd *cobra.Command, args []string) {
	sblog.Fatal("not implemented")
}
