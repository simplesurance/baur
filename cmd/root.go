package cmd

import (
	"fmt"
	"os"

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

var VerboseFlag bool

func initSb(_ *cobra.Command, _ []string) {
	sblog.EnableDebug(VerboseFlag)

}

func Execute() {
	rootCmd.PersistentFlags().BoolVarP(&VerboseFlag, "verbose", "v", false, "verbose output")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
