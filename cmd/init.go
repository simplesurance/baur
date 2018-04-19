package cmd

import (
	"os"
	"path"

	"github.com/simplesurance/sisubuild/cfg"
	"github.com/simplesurance/sisubuild/discover"
	"github.com/simplesurance/sisubuild/sblog"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "creates a new repository config in the current directory",
	Run:   initRepositoryCfg,
}

func initRepositoryCfg(cmd *cobra.Command, args []string) {
	repoRoot, err := discover.RepositoryRoot(cfg.RepositoryFile)
	if err == nil {
		sblog.Fatalf("repository configuration %s already exist",
			path.Join(repoRoot, cfg.RepositoryFile))
	}

	err = cfg.NewRepositoryFile(path.Join(repoRoot, cfg.RepositoryFile))
	if err != nil {
		if os.IsExist(err) {
			sblog.Fatalf("%s already exist", cfg.RepositoryFile)
		}

		sblog.Fatal(err)
	}

	sblog.Infof("written example repository configuration to %s",
		cfg.RepositoryFile)
}
