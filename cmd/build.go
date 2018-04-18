package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(buildCmd)
}

var buildCmd = &cobra.Command{
	Use:   "build <PATH>|<APP-NAME>|all",
	Short: "builds an application",
	Run:   printVersion,
	Args:  cobra.ExactArgs(1),
}

func printVersion(cmd *cobra.Command, args []string) {
	fmt.Println("not implemented")
	os.Exit(1)
}
