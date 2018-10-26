package command

import (
	"fmt"
	"os"
	"path"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur"
	"github.com/simplesurance/baur/cfg"
	"github.com/simplesurance/baur/log"
)

func init() {
	initCmd.AddCommand(initRepoCmd)
}

var initRepoCmd = &cobra.Command{
	Use:   "repo",
	Short: "create a repository config file in the current directory",
	Run:   initRepo,
}

func initRepo(cmd *cobra.Command, args []string) {
	rep, err := baur.FindRepository()
	if err == nil {
		log.Fatalf("repository configuration %s already exist",
			path.Join(rep.Path, baur.RepositoryCfgFile))
	}

	repCfg := cfg.ExampleRepository()

	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalln(err)
	}

	err = repCfg.ToFile(path.Join(cwd, baur.RepositoryCfgFile), false)
	if err != nil {
		if os.IsExist(err) {
			log.Fatalf("%s already exist\n", baur.RepositoryCfgFile)
		}

		log.Fatalln(err)
	}

	fmt.Printf("written example repository configuration to %s\n",
		baur.RepositoryCfgFile)
}
