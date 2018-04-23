package command

import (
	"os"

	"github.com/simplesurance/baur"
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

func mustFindRepository() *baur.Repository {
	sblog.Debug("searching for repository root...")

	rep, err := baur.FindRepository()
	if err != nil {
		if os.IsNotExist(err) {
			sblog.Fatalf("could not find repository root config file "+
				"ensure the file '%s' exist in the root",
				baur.RepositoryCfgFile)
		}

		sblog.Fatal(err)
	}

	sblog.Debugf("repository root found: %v", rep.Path)

	return rep
}

func initSb(_ *cobra.Command, _ []string) {
	sblog.EnableDebug(verboseFlag)
}

func Execute() {
	rootCmd.PersistentFlags().BoolVarP(&verboseFlag, "verbose", "v", false, "verbose output")

	if err := rootCmd.Execute(); err != nil {
		sblog.Fatal(err)
	}
}
