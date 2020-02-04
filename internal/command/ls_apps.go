package command

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur"
	"github.com/simplesurance/baur/format"
	"github.com/simplesurance/baur/format/csv"
	"github.com/simplesurance/baur/format/table"
	"github.com/simplesurance/baur/internal/command/flag"
	"github.com/simplesurance/baur/log"
	"github.com/simplesurance/baur/storage"
)

const (
	lsAppNameHeader        = "Name"
	lsAppNameParam         = "name"
	lsAppPathHeader        = "Path"
	lsAppPathParam         = "path"
	lsAppBuildStatusHeader = "Build Status"
	lsAppBuildStatusParam  = "build-status"
	lsAppBuildIDHeader     = "Build ID"
	lsAppBuildIDParam      = "build-id"
	lsAppGitCommitHeader   = "Git Commit"
	lsAppGitCommitParam    = "git-commit"
)

type lsAppsConf struct {
	csv         bool
	quiet       bool
	absPaths    bool
	buildStatus flag.BuildStatus
	fields      *flag.Fields
}

var lsAppsCmd = &cobra.Command{
	Use:   "apps [<APP-NAME>|<PATH>]...",
	Short: "list applications and their status",
	Run:   ls,
	Args:  cobra.ArbitraryArgs,
}

var lsAppsConfig lsAppsConf

func init() {
	lsAppsCmd.Flags().BoolVar(&lsAppsConfig.csv, "csv", false,
		"List applications in RFC4180 CSV format")

	lsAppsCmd.Flags().BoolVarP(&lsAppsConfig.quiet, "quiet", "q", false,
		"Suppress printing a header and progress dots")

	lsAppsCmd.Flags().BoolVar(&lsAppsConfig.absPaths, "abs-path", false,
		"Show absolute instead of relative paths")

	lsAppsCmd.Flags().VarP(&lsAppsConfig.buildStatus, "build-status", "s",
		lsAppsConfig.buildStatus.Usage(highlight))

	lsAppsConfig.fields = flag.NewFields([]string{
		lsAppNameParam,
		lsAppPathParam,
		lsAppBuildIDParam,
		lsAppBuildStatusParam,
		lsAppGitCommitParam,
	})
	lsAppsCmd.Flags().VarP(lsAppsConfig.fields, "fields", "f",
		lsAppsConfig.fields.Usage(highlight))

	lsCmd.AddCommand(lsAppsCmd)
}

func createHeader() []string {
	var headers []string

	for _, f := range lsAppsConfig.fields.Fields {
		switch f {
		case lsAppNameParam:
			headers = append(headers, lsAppNameHeader)
		case lsAppPathParam:
			headers = append(headers, lsAppPathHeader)
		case lsAppBuildStatusParam:
			headers = append(headers, lsAppBuildStatusHeader)
		case lsAppBuildIDParam:
			headers = append(headers, lsAppBuildIDHeader)
		case lsAppGitCommitParam:
			headers = append(headers, lsAppGitCommitHeader)
		default:
			panic(fmt.Sprintf("unsupported value '%v' in fields parameter", f))

		}
	}

	return headers
}

func ls(cmd *cobra.Command, args []string) {
	var headers []string
	var formatter format.Formatter
	var storageClt storage.Storer

	repo := MustFindRepository()
	apps := mustArgToApps(repo, args)
	writeHeaders := !lsAppsConfig.quiet && !lsAppsConfig.csv
	storageQueryNeeded := storageQueryIsNeeded()

	if storageQueryNeeded {
		storageClt = MustGetPostgresClt(repo)
	}

	if writeHeaders {
		headers = createHeader()
	}

	if lsAppsConfig.csv {
		formatter = csv.New(headers, os.Stdout)
	} else {
		formatter = table.New(headers, os.Stdout)
	}

	showProgress := len(apps) >= 5 && !lsAppsConfig.quiet && !lsAppsConfig.csv

	baur.SortAppsByName(apps)

	for i, app := range apps {
		var row []interface{}
		var build *storage.BuildWithDuration
		var buildStatus baur.BuildStatus

		if storageQueryNeeded {
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

		if lsAppsConfig.buildStatus.IsSet() && buildStatus != lsAppsConfig.buildStatus.Status {
			continue
		}

		row = assembleRow(app, build, buildStatus)

		if err := formatter.WriteRow(row); err != nil {
			log.Fatalln(err)
		}
	}

	if err := formatter.Flush(); err != nil {
		log.Fatalln(err)
	}
}

func storageQueryIsNeeded() bool {
	for _, f := range lsAppsConfig.fields.Fields {
		switch f {
		case lsAppBuildStatusParam:
			return true
		case lsAppBuildIDParam:
			return true
		case lsAppGitCommitParam:
			return true
		}
	}

	return false
}

func assembleRow(app *baur.App, build *storage.BuildWithDuration, buildStatus baur.BuildStatus) []interface{} {
	var row []interface{}

	for _, f := range lsAppsConfig.fields.Fields {
		switch f {
		case lsAppNameParam:
			row = append(row, app.Name)

		case lsAppPathParam:
			if lsAppsConfig.absPaths {
				row = append(row, app.Path)
			} else {
				row = append(row, app.RelPath)
			}

		case lsAppBuildStatusParam:
			row = append(row, buildStatus)

		case lsAppBuildIDParam:
			if buildStatus == baur.BuildStatusExist {
				row = append(row, fmt.Sprint(build.ID))
			} else {
				// no build exist, we don't have a build id
				row = append(row, "")
			}

		case lsAppGitCommitParam:
			if buildStatus == baur.BuildStatusExist {
				row = append(row, fmt.Sprint(build.VCSState.CommitID))
			} else {
				row = append(row, "")
			}
		}
	}

	return row
}
