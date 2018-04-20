package command

import (
	"os"
	"path"
	"strings"

	"github.com/simplesurance/baur/cfg"
	"github.com/simplesurance/baur/sblog"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(appInitCmd)
}

const appInitLongHelp = `
Create an application config file in the current directory.
The name parameter is set to the current directory name.`

var appInitCmd = &cobra.Command{
	Use:   "appinit",
	Short: "creates an application config file in the current directory",
	Long:  strings.TrimSpace(appInitLongHelp),
	Run:   appInit,
}

func appInit(cmd *cobra.Command, args []string) {
	mustFindRepositoryRoot()

	cwd, err := os.Getwd()
	if err != nil {
		sblog.Fatal(err)
	}
	appName := path.Base(cwd)

	appCfg := cfg.App{Name: appName}

	err = appCfg.ToFile(path.Join(cwd, cfg.AppFile))
	if err != nil {
		if os.IsExist(err) {
			sblog.Fatalf("%s already exist", cfg.AppFile)
		}

		sblog.Fatal(err)
	}

	sblog.Infof("written application configuration file to %s",
		cfg.AppFile)
}
