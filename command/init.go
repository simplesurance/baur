package command

import (
	"os"
	"path"

	"github.com/simplesurance/baur"
	"github.com/simplesurance/baur/cfg"
	"github.com/simplesurance/baur/log"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "creates a repository config in the current directory",
	Run:   initRepositoryCfg,
}

func initRepositoryCfg(cmd *cobra.Command, args []string) {
	rep, err := baur.FindRepository()
	if err == nil {
		log.Fatalf("repository configuration %s already exist\n",
			path.Join(rep.Path, baur.RepositoryCfgFile))
	}

	repCfg := cfg.ExampleRepository()

	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalln(err)
	}

	err = repCfg.ToFile(path.Join(cwd, baur.RepositoryCfgFile))
	if err != nil {
		if os.IsExist(err) {
			log.Fatalf("%s already exist\n", baur.RepositoryCfgFile)
		}

		log.Fatalln(err)
	}

	log.Infof("written example repository configuration to %s\n",
		baur.RepositoryCfgFile)
}
