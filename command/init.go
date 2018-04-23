package command

import (
	"os"
	"path"

	"github.com/simplesurance/baur"
	"github.com/simplesurance/baur/cfg"
	"github.com/simplesurance/baur/discover"
	"github.com/simplesurance/baur/sblog"
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
	repoRoot, err := discover.RepositoryRoot(baur.RepositoryCfgFile)
	if err == nil {
		sblog.Fatalf("repository configuration %s already exist",
			path.Join(repoRoot, baur.RepositoryCfgFile))
	}

	repCfg := cfg.ExampleRepository()
	err = repCfg.ToFile(path.Join(repoRoot, baur.RepositoryCfgFile))
	if err != nil {
		if os.IsExist(err) {
			sblog.Fatalf("%s already exist", baur.RepositoryCfgFile)
		}

		sblog.Fatal(err)
	}

	sblog.Infof("written example repository configuration to %s",
		baur.RepositoryCfgFile)
}
