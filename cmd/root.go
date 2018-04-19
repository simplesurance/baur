package cmd

import (
	"github.com/simplesurance/baur/sblog"
	"github.com/simplesurance/baur/version"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:              "baur",
	Short:            "baur manages builds and artifacts in mono repositories.",
	Version:          version.FullVerNr(),
	PersistentPreRun: initSb,
}

var verboseFlag bool

func initSb(_ *cobra.Command, _ []string) {
	sblog.EnableDebug(verboseFlag)
}

func Execute() {
	rootCmd.PersistentFlags().BoolVarP(&verboseFlag, "verbose", "v", false, "verbose output")

	if err := rootCmd.Execute(); err != nil {
		sblog.Fatal(err)
	}
}
