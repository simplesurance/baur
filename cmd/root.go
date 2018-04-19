package cmd

import (
	"github.com/simplesurance/sisubuild/sblog"
	"github.com/simplesurance/sisubuild/version"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:              "sb",
	Short:            "sisubuild manages builds and artifacts in mono repositories.",
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
