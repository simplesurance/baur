package command

import (
	"os"
	"path"
	"strings"

	"github.com/simplesurance/baur"
	"github.com/simplesurance/baur/cfg"
	"github.com/simplesurance/baur/log"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(appInitCmd)
}

const appInitLongHelp = `
Create an application config file in the current directory.
If no name is passed, the application name will be the name of the current directory.`

const appInitExample = `
baur appinit shop-ui    create a new application config with the app name set to shop-ui`

var appInitCmd = &cobra.Command{
	Use:     "appinit [APP-NAME]",
	Short:   "creates an application config file in the current directory",
	Long:    strings.TrimSpace(appInitLongHelp),
	Example: strings.TrimSpace(appInitExample),
	Run:     appInit,
	Args:    cobra.MaximumNArgs(1),
}

func appInit(cmd *cobra.Command, args []string) {
	var appName string
	mustFindRepository()

	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalln(err)
	}

	if len(args) > 0 {
		appName = args[0]
	} else {
		appName = path.Base(cwd)
	}

	appCfg := cfg.ExampleApp(appName)

	err = appCfg.ToFile(path.Join(cwd, baur.AppCfgFile))
	if err != nil {
		if os.IsExist(err) {
			log.Fatalf("%s already exist\n", baur.AppCfgFile)
		}

		log.Fatalln(err)
	}

	log.Infof("configuration file for %s was written to %s\n",
		appName, baur.AppCfgFile)
}
