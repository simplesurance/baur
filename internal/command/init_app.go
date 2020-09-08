package command

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur"
	"github.com/simplesurance/baur/cfg"
	"github.com/simplesurance/baur/log"
)

func init() {
	initCmd.AddCommand(initAppCmd)
}

const initAppLongHelp = `
Create an application config file in the current directory.
If no name is passed, the application name will be the name of the current directory.`

const initAppExample = `
baur init app shop-ui	create an application config with the app name set to shop-ui`

var initAppCmd = &cobra.Command{
	Use:     "app [APP-NAME]",
	Short:   "creates an application config file in the current directory",
	Long:    strings.TrimSpace(initAppLongHelp),
	Example: strings.TrimSpace(initAppExample),
	Run:     initApp,
	Args:    cobra.MaximumNArgs(1),
}

func initApp(cmd *cobra.Command, args []string) {
	var appName string
	MustFindRepository()

	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalln(err)
	}

	if len(args) > 0 {
		appName = args[0]
	} else {
		appName = filepath.Base(cwd)
	}

	appCfg := cfg.ExampleApp(appName)

	err = appCfg.ToFile(filepath.Join(cwd, baur.AppCfgFile))
	if err != nil {
		if os.IsExist(err) {
			log.Fatalf("%s already exist\n", baur.AppCfgFile)
		}

		log.Fatalln(err)
	}

	fmt.Printf("Application configuration file was written to %s\n",
		highlight(baur.AppCfgFile))
}
