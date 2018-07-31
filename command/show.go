package command

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/simplesurance/baur"
	"github.com/simplesurance/baur/log"
	"github.com/simplesurance/baur/storage"
	"github.com/spf13/cobra"
)

var showPrintPathOnly bool

func init() {
	showCmd.Flags().BoolVar(&showPrintPathOnly, "path-only", false, "only print the path of the application")
	rootCmd.AddCommand(showCmd)
}

const showExampleHelp = `
baur show claim-service		        show informations about the claim-service application
baur show 482				show informations about the build with ID 482
baur show --path-only claim-service	show the path of the directory of the claim-service application
baur show .		                show informations about the application in the current directory
baur show		                show informations about the repository
`

var showCmd = &cobra.Command{
	Use:     "show [<APP-NAME>|<PATH>|<BUILD-ID>]",
	Short:   "shows informations about applications and builds",
	Example: strings.TrimSpace(showExampleHelp),
	Run:     show,
	Args:    cobra.MaximumNArgs(1),
}

func showRepositoryInformation(rep *baur.Repository) {
	if showPrintPathOnly {
		fmt.Println(rep.Path)
		return
	}

	fmt.Printf("# Repository Information\n")
	fmt.Printf("Root:\t\t%s\n", rep.Path)
	fmt.Printf("PostgreSQL URL:\t%s\n", rep.PSQLURL)
}

func showApplicationInformation(app *baur.App) {
	if showPrintPathOnly {
		fmt.Println(app.Path)
		return
	}

	fmt.Printf("# Application Information\n")
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 8, ' ', 0)

	fmt.Fprintf(tw, "Name:\t%s\n", app.Name)
	fmt.Fprintf(tw, "Directory:\t%s\n", app.RelPath)
	fmt.Fprintf(tw, "Build Command:\t%s\n", app.BuildCmd)

	if len(app.Outputs) == 0 {
		fmt.Fprintf(tw, "Outputs: -None-\n")
	} else {
		fmt.Fprintf(tw, "Outputs:\t\n")
		for i, art := range app.Outputs {
			fmt.Fprintf(tw, "\tLocal:\t%s\n", art.String())
			fmt.Fprintf(tw, "\tRemote:\t%s\n", art.UploadDestination())
			if i < len(app.Outputs)-1 {
				fmt.Fprintf(tw, "\t\t\n")
			}
		}
	}

	tw.Flush()
}

func showBuildInformation(rep *baur.Repository, buildID int) {
	clt := mustGetPostgresClt(rep)
	build, err := clt.GetBuildWithoutInputs(buildID)
	if err != nil {
		if err == storage.ErrNotExist {
			log.Fatalf("build with id %d does not exist\n", buildID)
		}

		log.Fatalln("querying datatbase failed:", err)
	}

	fmt.Printf("# Build Information\n")
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 8, ' ', 0)

	fmt.Fprintf(tw, "Application:\t%s\n", build.AppName)
	fmt.Fprintf(tw, "Build ID:\t%d\n", buildID)

	fmt.Fprintf(tw, "Build Started at:\t%s\n", build.StartTimeStamp)
	fmt.Fprintf(tw, "Build Duration:\t%s\n",
		durationToStrSec(build.StopTimeStamp.Sub(build.StartTimeStamp)))

	fmt.Fprintf(tw, "Git Commit:\t%s\n", vcsStr(&build.VCSState))

	fmt.Fprintf(tw, "Input Digest:\t%s\n", build.TotalInputDigest)

	fmt.Fprintf(tw, "Outputs:\t\n")
	for i, o := range build.Outputs {
		fmt.Fprintf(tw, "\tURL:\t%s\n", o.URL)
		fmt.Fprintf(tw, "\tDigest:\t%s\n", o.Digest)
		fmt.Fprintf(tw, "\tSize:\t%.3fMiB\n", float64(o.SizeBytes)/1024/1024)
		fmt.Fprintf(tw, "\tUpload Duration:\t%s\n", durationToStrSec(o.UploadDuration))

		if i < (len(build.Outputs) - 1) {
			fmt.Fprintf(tw, "\t\t\t\n")
		}
	}

	tw.Flush()

}

func show(cmd *cobra.Command, args []string) {
	rep := mustFindRepository()

	if len(args) == 0 {
		showRepositoryInformation(rep)

		os.Exit(0)
	}

	buildID, err := strconv.ParseInt(args[0], 10, 32)
	if err == nil {
		showBuildInformation(rep, int(buildID))
		os.Exit(0)
	}

	app := mustArgToApp(rep, args[0])
	showApplicationInformation(app)
}
