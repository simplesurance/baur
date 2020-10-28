package command

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur/v1"
	"github.com/simplesurance/baur/v1/cfg"
	"github.com/simplesurance/baur/v1/internal/command/term"
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
	Short:   "create an application config file in the current directory",
	Long:    strings.TrimSpace(initAppLongHelp),
	Example: strings.TrimSpace(initAppExample),
	Run:     initApp,
	Args:    cobra.MaximumNArgs(1),
}

func initApp(cmd *cobra.Command, args []string) {
	var appName string

	cwd, err := os.Getwd()
	exitOnErr(err)

	if len(args) > 0 {
		appName = args[0]
	} else {
		appName = filepath.Base(cwd)
	}

	appCfg := cfg.ExampleApp(appName)

	err = appCfg.ToFile(filepath.Join(cwd, baur.AppCfgFile), cfg.ToFileOptCommented())
	if err != nil {
		if os.IsExist(err) {
			stderr.Printf("%s already exist\n", baur.AppCfgFile)
			exitFunc(1)
		}

		stderr.Println(err)
		exitFunc(1)
	}

	stdout.Printf("Application configuration file was written to %s\n",
		term.Highlight(baur.AppCfgFile))
}
