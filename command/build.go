package command

import (
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/simplesurance/baur"
	"github.com/simplesurance/baur/log"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(buildCmd)
}

const buildLongHelp = `
Builds applications.
If no argument is the application in the current directory is build.
If the current directory does not contain an application, all applications are build.`

const buildExampleHelp = `
baur build all		      build all applications in the repository
baur build payment-service    build the application with the name payment-service
baur build ui/shop	      build the application in the directory ui/shop`

var buildCmd = &cobra.Command{
	Use:     "build [<PATH>|<APP-NAME>|all]",
	Short:   "builds an application",
	Long:    strings.TrimSpace(buildLongHelp),
	Run:     build,
	Example: strings.TrimSpace(buildExampleHelp),
	Args:    cobra.MaximumNArgs(1),
}

func isAppDir(arg string) bool {
	cfgPath := path.Join(arg, baur.AppCfgFile)
	_, err := os.Stat(cfgPath)
	if err == nil {
		return true
	}

	return false
}

func mustArgToApps(repo *baur.Repository, arg string) []*baur.App {
	if strings.ToLower(arg) == "all" {
		apps, err := repo.FindApps()
		if err != nil {
			log.Fatalln(err)
		}

		return apps
	}

	if isAppDir(arg) {
		app, err := repo.AppByDir(arg)
		if err != nil {
			log.Fatalf("could not find application in dir '%s': %s\n", arg, err)
		}

		return []*baur.App{app}
	}

	app, err := repo.AppByName(arg)
	if err != nil {
		log.Fatalf("could not find application with name '%s': %s\n", arg, err)
	}

	return []*baur.App{app}
}

func longestAppName(apps []*baur.App) int {
	result := 0

	for _, app := range apps {
		if len(app.Name) > result {
			result = len(app.Name)
		}
	}
	return result
}

func build(cmd *cobra.Command, args []string) {
	var apps []*baur.App
	var totalBuilDuration time.Duration

	repo := mustFindRepository()

	if len(args) > 0 {
		apps = mustArgToApps(repo, args[0])
	} else if isAppDir(".") {
		apps = mustArgToApps(repo, ".")
	} else {
		apps = mustArgToApps(repo, "all")
	}

	baur.SortAppsByName(apps)

	if !verboseFlag {
		colLen := 20
		maxColLen := 60
		maxAppNameLen := longestAppName(apps)

		if maxAppNameLen > colLen {
			colLen = maxAppNameLen
		}
		if colLen > maxColLen {
			colLen = maxColLen
		}

		fmt.Printf("%-*s\t%-*s\t%-*s\n",
			colLen, "# Application",
			colLen, "Status",
			colLen, "Duration")
		for _, app := range apps {
			fmt.Printf("%-*s\t", colLen, app.Name)

			res, err := app.Build()
			if err != nil {
				fmt.Printf("%-*s\n\n", colLen, "error")
				log.Fatalln(err)
			}

			if !res.Success {
				fmt.Printf("%-*s\n\n", colLen, "failed")
				log.Fatalf("build command (%s) exited with code: %d, Output:\n%s\n",
					app.BuildCmd, res.ExitCode, res.Output)
			}

			fmt.Printf("%-*s\t", colLen, "success")
			fmt.Printf("%-*s\n", colLen, res.Duration)

			totalBuilDuration += res.Duration
		}

		log.Infof("\ntotal build duration: %s\n", totalBuilDuration)
		os.Exit(0)
	}

	for _, app := range apps {
		log.Infof("building %s\n", app.Name)

		res, err := app.Build()
		if err != nil {
			log.Fatalln(err)
		}
		if !res.Success {
			log.Fatalf("build failed, command (%s) exited with code: %d\n",
				app.BuildCmd, res.ExitCode)
		}

		log.Infof("build finished successfully in %s\n\n", res.Duration)
		totalBuilDuration += res.Duration
	}

	log.Infof("total build duration: %s\n", totalBuilDuration)
}
