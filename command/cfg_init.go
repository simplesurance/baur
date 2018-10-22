package command

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur"
	"github.com/simplesurance/baur/cfg"
	"github.com/simplesurance/baur/log"
)

func init() {
	cfgCmd.AddCommand(cfgInitCmd)
}

const cfgInitCmdHelp = `
Create a template application or repository config file in the current directory.
`

const appsInitExample = `
baur cfg init shop-ui    create a .app.toml application config with the app name set to shop-ui
baur cfg init		 create a .baur.toml repository application config`

var cfgInitCmd = &cobra.Command{
	Use:     "init [APP-NAME]",
	Short:   "creates an application config file in the current directory",
	Long:    strings.TrimSpace(appsInitLongHelp),
	Example: strings.TrimSpace(appsInitExample),
	Run:     appsInit,
	Args:    cobra.MaximumNArgs(1),
}

func appsInit(cmd *cobra.Command, args []string) {
	var appName string
	MustFindRepository()

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

	fmt.Printf("configuration file for %s was written to %s\n",
		appName, baur.AppCfgFile)
}
