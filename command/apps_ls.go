package command

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur"
	"github.com/simplesurance/baur/command/flag"
	"github.com/simplesurance/baur/format"
	"github.com/simplesurance/baur/format/csv"
	"github.com/simplesurance/baur/format/table"
	"github.com/simplesurance/baur/log"
	"github.com/simplesurance/baur/storage"
)

const (
	lsAppNameHeader        = "Name"
	lsAppNameParam         = "name"
	lsAppPathHeader        = "Path"
	lsAppPathParam         = "path"
	lsAppBuildIDHeader     = "Build ID"
	lsAppBuildIDParam      = "build-id"
	lsAppBuildStatusHeader = "Build Status"
	lsAppBuildStatusParam  = "build-status"
)

type appsLsConf struct {
	csv         bool
	quiet       bool
	absPaths    bool
	buildStatus flag.BuildStatus
	fields      *flag.Fields
}

var appsLsCmd = &cobra.Command{
	Use:   "ls [<APP-NAME>]...",
	Short: "list applications and their status",
	Run:   ls,
	Args:  cobra.ArbitraryArgs,
}

var appsLsConfig appsLsConf

func init() {
	appsLsCmd.Flags().BoolVar(&appsLsConfig.csv, "csv", false,
		"List applications in RFC4180 CSV format")

	appsLsCmd.Flags().BoolVarP(&appsLsConfig.quiet, "quiet", "q", false,
		"Only print application names")

	appsLsCmd.Flags().BoolVar(&appsLsConfig.absPaths, "abs-path", false,
		"Show absolute instead of relative paths")

	appsLsCmd.Flags().VarP(&appsLsConfig.buildStatus, "build-status", "s",
		appsLsConfig.buildStatus.Usage(highlight))

	appsLsConfig.fields = flag.NewFields([]string{
		lsAppNameParam,
		lsAppPathParam,
		lsAppBuildIDParam,
		lsAppBuildStatusParam,
	})
	appsLsCmd.Flags().VarP(appsLsConfig.fields, "fields", "f",
		appsLsConfig.fields.Usage(highlight))

	appsCmd.AddCommand(appsLsCmd)
}

func createHeader() []string {
	var headers []string

	if appsLsConfig.fields.IsSet(lsAppNameParam) {
		headers = append(headers, lsAppNameHeader)
	}

	if appsLsConfig.fields.IsSet(lsAppPathParam) {
		headers = append(headers, lsAppPathHeader)
	}

	if appsLsConfig.fields.IsSet(lsAppBuildIDParam) {
		headers = append(headers, lsAppBuildIDHeader)
	}

	if appsLsConfig.fields.IsSet(lsAppBuildStatusParam) {
		headers = append(headers, lsAppBuildStatusHeader)
	}

	return headers
}

func ls(cmd *cobra.Command, args []string) {
	var headers []string
	var formatter format.Formatter
	var storageClt storage.Storer

	repo := MustFindRepository()
	apps := mustArgToApps(repo, args)
	writeHeaders := !appsLsConfig.quiet && !appsLsConfig.csv

	if storageQueryIsNeeded() {
		storageClt = MustGetPostgresClt(repo)
	}

	if writeHeaders {
		headers = createHeader()
	}

	if appsLsConfig.csv {
		formatter = csv.New(headers, os.Stdout, writeHeaders)
	} else {
		formatter = table.New(headers, os.Stdout, writeHeaders)
	}

	showProgress := len(apps) >= 5 && !appsLsConfig.quiet && !appsLsConfig.csv

	for i, app := range apps {
		var row []interface{}
		var build *storage.Build
		var buildStatus baur.BuildStatus

		if storageQueryIsNeeded() {
			var err error

			buildStatus, build, err = baur.GetBuildStatus(storageClt, app)
			if err != nil {
				log.Fatalf("gathering informations for %s failed: %s", app, err)
			}

			// querying the build status for all applications can
			// take some time, output progress dots to let the user
			// know that something is happening
			if showProgress {
				fmt.Printf(".")

				if i+1 == len(apps) {
					fmt.Printf("\n\n")
				}
			}
		}

		if appsLsConfig.buildStatus.IsSet() && buildStatus != appsLsConfig.buildStatus.Status {
			continue
		}

		if appsLsConfig.quiet {
			row = assembleQuietRow(app)
		} else {
			row = assembleRow(app, build, buildStatus)
		}

		if err := formatter.WriteRow(row); err != nil {
			log.Fatalln(err)
		}
	}

	formatter.Flush()
}

func assembleQuietRow(app *baur.App) []interface{} {
	return []interface{}{app.Name}
}

func storageQueryIsNeeded() bool {
	return (appsLsConfig.buildStatus.IsSet() ||
		appsLsConfig.fields.IsSet(lsAppBuildIDParam) ||
		appsLsConfig.fields.IsSet(lsAppBuildStatusParam))
}

func assembleRow(app *baur.App, build *storage.Build, buildStatus baur.BuildStatus) []interface{} {
	var row []interface{}

	if appsLsConfig.fields.IsSet(lsAppNameParam) {
		row = append(row, app.Name)
	}

	if appsLsConfig.fields.IsSet(lsAppPathParam) {
		if appsLsConfig.absPaths {
			row = append(row, app.Path)
		} else {
			row = append(row, app.RelPath)
		}
	}

	if appsLsConfig.fields.IsSet(lsAppBuildIDParam) {
		if buildStatus == baur.BuildStatusExist {
			row = append(row, fmt.Sprint(build.ID))
		} else {
			// no build exist, we don't have a build id
			row = append(row, "")
		}
	}

	if appsLsConfig.fields.IsSet(lsAppBuildStatusParam) {
		row = append(row, colorizedBuildStatus((buildStatus)))
	}

	return row
}

func colorizedBuildStatus(status baur.BuildStatus) string {
	switch status {
	case baur.BuildStatusExist:
		return greenHighlight(baur.BuildStatusExist.String())

	case baur.BuildStatusOutstanding:
		return redHighlight(baur.BuildStatusOutstanding.String())

	case baur.BuildStatusInputsUndefined:
		return yellowHighlight(baur.BuildStatusInputsUndefined.String())
	default:
		panic(fmt.Sprintf("invalid build-status: %v", status))
	}
}
