package command

import (
	"os"

	"github.com/simplesurance/baur"
	"github.com/simplesurance/baur/log"
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
	log.Debugln("searching for repository root...")

	rep, err := baur.FindRepository()
	if err != nil {
		if os.IsNotExist(err) {
			log.Fatalf("could not find repository root config file "+
				"ensure the file '%s' exist in the root\n",
				baur.RepositoryCfgFile)
		}

		log.Fatalln(err)
	}

	log.Debugf("repository root found: %v\n", rep.Path)

	return rep
}

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
