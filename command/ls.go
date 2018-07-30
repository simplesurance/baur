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
	lsCmd.Flags().BoolVarP(&lsShowAbsPath, "abs-path", "a", false,
		"show absolute instead of relative paths")
	rootCmd.AddCommand(lsCmd)
}

func appPath(a *baur.App, absolutePaths bool) string {
	if absolutePaths {
		return a.Path
	}

	return a.RelPath
}

func lsPlain(apps []*baur.App) {
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 8, ' ', 0)

	fmt.Fprintf(tw, "# Name\tDirectory\n")

	for _, app := range apps {
		path := appPath(app, lsShowAbsPath)
		fmt.Fprintf(tw, "%s\t%s\n", app.Name, path)
	}

	tw.Flush()

	term.PrintSep()
	fmt.Printf("Total: %v\n", len(apps))
}

func lsBuildStatusPlain(apps []*baur.App, storage storage.Storer) {
	var buildExist int
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 8, ' ', 0)

	fmt.Fprintf(tw, "# Name\tBuild Status\tBuild ID\tGit Commit\n")

	for _, app := range apps {
		buildStatus, build, buildID := mustGetBuildStatus(app, storage)

		if buildStatus == baur.BuildStatusExist {
			buildExist++
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", app.Name, buildStatus, buildID, vcsStr(&build.VCSState))
			continue
		}

		fmt.Fprintf(tw, "%s\t%s\t\t\n", app.Name, buildStatus)
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

func lsBuildStatusCSV(apps []*baur.App, storage storage.Storer) {
	csvw := csv.NewWriter(os.Stdout)

	for _, app := range apps {
		buildStatus, build, buildID := mustGetBuildStatus(app, storage)

		if buildStatus == baur.BuildStatusExist {
			csvw.Write([]string{
				app.Name,
				buildStatus.String(),
				buildID,
				vcsStr(&build.VCSState),
			})

			continue
		}

		csvw.Write([]string{
			app.Name,
			buildStatus.String(),
			buildID,
		})

	}

	csvw.Flush()
}

func lsCSV(apps []*baur.App) {
	csvw := csv.NewWriter(os.Stdout)

	for _, app := range apps {
		path := appPath(app, lsShowAbsPath)

		csvw.Write([]string{app.Name, path})
	}

	csvw.Flush()
}

func ls(cmd *cobra.Command, args []string) {
	var storage storage.Storer
	rep := mustFindRepository()
	apps := mustFindApps(rep)

	baur.SortAppsByName(apps)

	if lsShowBuildStatus {
		storage = mustGetPostgresClt(rep)

		if lsCSVFmt {
			lsBuildStatusCSV(apps, storage)
			os.Exit(0)
		}

		lsBuildStatusPlain(apps, storage)
		os.Exit(0)
	}

	if lsCSVFmt {
		lsCSV(apps)
		os.Exit(0)
	}

	lsPlain(apps)
}

func mustGetBuildStatus(app *baur.App, storage storage.Storer) (baur.BuildStatus, *storage.Build, string) {
	var strBuildID string

	status, id, build, err := baur.GetBuildStatus(storage, app)
	if err != nil {
		log.Fatalln("evaluating build status failed:", err)
	}

	if id != -1 {
		strBuildID = fmt.Sprint(id)
	}

	return status, build, strBuildID
}
