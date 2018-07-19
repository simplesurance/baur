package command

import (
	"encoding/csv"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/simplesurance/baur"
	"github.com/simplesurance/baur/log"
	"github.com/simplesurance/baur/storage"
	"github.com/simplesurance/baur/term"
	"github.com/spf13/cobra"
)

var lsCSVFmt bool
var lsShowBuildStatus bool
var lsShowAbsPath bool

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "list all applications in the repository",
	Run:   ls,
}

func init() {
	lsCmd.Flags().BoolVar(&lsCSVFmt, "csv", false, "list applications in RFC4180 CSV format")
	lsCmd.Flags().BoolVarP(&lsShowBuildStatus, "build-status", "b", false,
		"shows if a build for the application exist")
	lsCmd.Flags().BoolVarP(&lsShowAbsPath, "abs-paths", "a", false,
		"show absolute instead of relative paths")
	rootCmd.AddCommand(lsCmd)
}

func appPath(a *baur.App, absolutePaths bool) string {
	if absolutePaths {
		return a.Path
	}

	return a.RelPath
}

func lsPlain(apps []*baur.App, storage storage.Storer) {
	var buildExist int
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 8, ' ', 0)

	if lsShowBuildStatus {
		fmt.Fprintf(tw, "# Name\tDirectory\tBuild Status\n")
	} else {
		fmt.Fprintf(tw, "# Name\tDirectory\n")
	}

	for _, app := range apps {
		path := appPath(app, lsShowAbsPath)

		if lsShowBuildStatus {
			buildStatus, buildID := mustGetBuildStatus(app, storage)

			if buildStatus == baur.BuildStatusExist {
				fmt.Fprintf(tw, "%s\t%s\t%s (ID: %s)\n",
					app.Name, path, buildStatus, buildID)
				buildExist++

			} else {
				fmt.Fprintf(tw, "%s\t%s\t%s\n",
					app.Name, path, buildStatus)
			}

			continue
		}

		fmt.Fprintf(tw, "%s\t%s\n", app.Name, path)
	}

	tw.Flush()

	term.PrintSep()

	if lsShowBuildStatus {
		fmt.Printf("Total: %d\n", len(apps))
		fmt.Printf("Outstanding builds: %d\n", len(apps)-buildExist)

		return
	}

	fmt.Printf("Total: %v\n", len(apps))
}

func lsCSV(apps []*baur.App, storage storage.Storer) {
	csvw := csv.NewWriter(os.Stdout)

	for _, app := range apps {
		path := appPath(app, lsShowAbsPath)

		if lsShowBuildStatus {
			buildStatus, buildID := mustGetBuildStatus(app, storage)

			csvw.Write([]string{
				app.Name,
				path,
				buildStatus.String(),
				buildID,
			})

			continue
		}

		csvw.Write([]string{app.Name, path})
	}

	csvw.Flush()
}

func ls(cmd *cobra.Command, args []string) {
	var storage storage.Storer
	rep := mustFindRepository()
	apps := mustFindApps(rep)

	if lsShowBuildStatus {
		storage = mustGetPostgresClt(rep)
	}

	baur.SortAppsByName(apps)

	if lsCSVFmt {
		lsCSV(apps, storage)
		os.Exit(0)
	}

	lsPlain(apps, storage)
}

func mustGetBuildStatus(app *baur.App, storage storage.Storer) (baur.BuildStatus, string) {
	var strBuildID string

	status, id, err := baur.GetBuildStatus(storage, app)
	if err != nil {
		log.Fatalln("evaluating build status failed:", err)
	}

	if id != -1 {
		strBuildID = fmt.Sprint(id)
	}

	return status, strBuildID
}
