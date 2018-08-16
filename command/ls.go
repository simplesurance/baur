package command

import (
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"text/tabwriter"

	"github.com/simplesurance/baur"
	"github.com/simplesurance/baur/log"
	"github.com/simplesurance/baur/storage"
	"github.com/simplesurance/baur/term"
	"github.com/spf13/cobra"
)

var (
	lsCSVFmt, lsShowBuildStatus, lsShowAbsPath bool
	lsWantedStatus                             string
)

const (
	lsNameCol   string = "Name"
	lsStatusCol        = "Build Status"
	lsIDCol            = "Build ID"
	lsVCSCol           = "Git Commit"
)

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
	lsCmd.Flags().StringVarP(&lsWantedStatus, "status", "s", "",
		"filter the list to a specific status")
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

func longestAppNameLen(apps []*baur.App) int {
	var longest int

	for _, a := range apps {
		if len(a.Name) > longest {
			longest = len(a.Name)
		}
	}

	return longest
}

func longestStrLen(strs ...string) int {
	var longest int

	for _, s := range strs {
		if len(s) > longest {
			longest = len(s)
		}
	}

	return longest
}

func max(a, b int) int {
	if a > b {
		return a
	}

	return b
}

func lsBuildStatusPlain(apps []*baur.App, storage storage.Storer, wantedStatus baur.Status) {
	const sepSpaces = 2
	var (
		buildExist int

		nameColLen   = max(longestAppNameLen(apps), len(lsNameCol)) + sepSpaces
		statusColLen = max(
			longestStrLen(baur.BuildStatusExist.String(), baur.BuildStatusInputsUndefined.String(), baur.BuildStatusOutstanding.String()),
			len(lsStatusCol),
		) + sepSpaces
		idColLen  = max(len(string(math.MaxInt64)), len(lsIDCol)) + sepSpaces
		vcsColLen = max(40+len("-dirty"), len(lsVCSCol))
	)

	if nameColLen <= 2 {
		nameColLen = 6
	}

	fmt.Printf("# %-*s\t%-*s\t%-*s\t%-*s\n",
		nameColLen-2, lsNameCol,
		statusColLen, lsStatusCol,
		idColLen, lsIDCol,
		vcsColLen, lsVCSCol)

	skippedCount := 0

	for _, app := range apps {
		buildStatus, build, buildID := mustGetBuildStatus(app, storage)

		if wantedStatus.IsNotNull() && wantedStatus != buildStatus {
			skippedCount++
			continue
		}

		if buildStatus == baur.BuildStatusExist {
			buildExist++
			fmt.Printf("%-*s\t%-*s\t%-*s\t%-*s\n", nameColLen, app.Name,
				statusColLen, buildStatus,
				idColLen, buildID,
				vcsColLen, vcsStr(&build.VCSState))
			continue
		}

		fmt.Printf("%-*s\t%-*s\t\t\n", nameColLen, app.Name, statusColLen, buildStatus)
	}

	term.PrintSep()

	if lsShowBuildStatus {
		fmt.Printf("Total: %d\n", len(apps))
		fmt.Printf("Outstanding builds: %d\n", len(apps)-buildExist)

		if skippedCount > 0 {
			fmt.Printf("Skipped: %d\n", skippedCount)
		}

		return
	}

	fmt.Printf("Total: %v\n", len(apps))
}

func lsBuildStatusCSV(apps []*baur.App, storage storage.Storer, wantedStatus baur.Status) {
	csvw := csv.NewWriter(os.Stdout)

	for _, app := range apps {
		buildStatus, build, buildID := mustGetBuildStatus(app, storage)

		if wantedStatus.IsNotNull() && wantedStatus != buildStatus {
			continue
		}

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
	var wantedStatus baur.Status

	if lsWantedStatus != "" {
		var err error
		wantedStatus, err = baur.NewAppStatus(lsWantedStatus)
		if err != nil {
			log.Fatalln(err)
		}
	}

	var storage storage.Storer
	rep := mustFindRepository()
	apps := mustFindApps(rep)

	baur.SortAppsByName(apps)

	if lsShowBuildStatus {
		storage = mustGetPostgresClt(rep)

		if lsCSVFmt {
			lsBuildStatusCSV(apps, storage, wantedStatus)
			os.Exit(0)
		}

		lsBuildStatusPlain(apps, storage, wantedStatus)
		os.Exit(0)
	}

	if lsCSVFmt {
		lsCSV(apps)
		os.Exit(0)
	}

	if wantedStatus > 0 {
		log.Fatalln("Can't filter a plain list by status")
	}

	lsPlain(apps)
}

func mustGetBuildStatus(app *baur.App, storage storage.Storer) (baur.Status, *storage.Build, string) {
	var strBuildID string

	status, build, err := baur.GetAppStatus(storage, app)
	if err != nil {
		log.Fatalf("evaluating build status of %s failed: %s\n", app, err)
	}

	if build != nil {
		strBuildID = fmt.Sprint(build.ID)
	}

	app.Status = &status

	return status, build, strBuildID
}
