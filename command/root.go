package command

import (
	"github.com/simplesurance/baur/log"
	"github.com/simplesurance/baur/version"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:              "baur",
	Short:            "baur manages builds and artifacts in mono repositories.",
	Version:          version.CurSemVer.String(),
	PersistentPreRun: initSb,
}

var verboseFlag bool

func initSb(_ *cobra.Command, _ []string) {
	log.DebugEnabled = verboseFlag
}

// Execute parses commandline flags and execute their actions
func Execute() {
	rootCmd.PersistentFlags().BoolVarP(&verboseFlag, "verbose", "v", false, "verbose output")

	if err := rootCmd.Execute(); err != nil {
		log.Fatalln(err)
	}
}
