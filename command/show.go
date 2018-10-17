package command

import (
	"fmt"
	"os"
	"strconv"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur"
	"github.com/simplesurance/baur/log"
	"github.com/simplesurance/baur/storage"
)

var showPrintPathOnly bool

func init() {
	showCmd.Flags().BoolVar(&showPrintPathOnly, "path-only", false, "only print the path of the application")
	rootCmd.AddCommand(showCmd)
}

var showCmd = &cobra.Command{
	Use:   "show [<APP-NAME>|<PATH>|<BUILD-ID>]",
	Short: "shows informations about applications and builds",
	Run:   showOld,
	Args:  cobra.MaximumNArgs(1),
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
	clt := MustGetPostgresClt(rep)
	build, err := clt.GetBuildWithoutInputs(buildID)
	if err != nil {
		if err == storage.ErrNotExist {
			log.Fatalf("build with id %d does not exist\n", buildID)
		}

		log.Fatalln("querying datatbase failed:", err)
	}

	fmt.Printf("# Build Information\n")
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 8, ' ', 0)

	fmt.Fprintf(tw, "Application:\t%s\n", build.Application.Name)
	fmt.Fprintf(tw, "Build ID:\t%d\n", buildID)

	fmt.Fprintf(tw, "Build Started at:\t%s\n", build.StartTimeStamp)
	fmt.Fprintf(tw, "Build Duration:\t%s\n",
		durationToStrSec(build.StopTimeStamp.Sub(build.StartTimeStamp)))

	fmt.Fprintf(tw, "Git Commit:\t%s\n", vcsStr(&build.VCSState))

	fmt.Fprintf(tw, "Input Digest:\t%s\n", build.TotalInputDigest)

	fmt.Fprintf(tw, "Outputs:\t\n")
	for i, o := range build.Outputs {
		fmt.Fprintf(tw, "\tURL:\t%s\n", o.Upload.URL)
		fmt.Fprintf(tw, "\tDigest:\t%s\n", o.Digest)
		fmt.Fprintf(tw, "\tSize:\t%.3fMiB\n", float64(o.SizeBytes)/1024/1024)
		fmt.Fprintf(tw, "\tUpload Duration:\t%s\n", durationToStrSec(o.Upload.UploadDuration))

		if i < (len(build.Outputs) - 1) {
			fmt.Fprintf(tw, "\t\t\t\n")
		}
	}

	tw.Flush()

}

func showOld(cmd *cobra.Command, args []string) {
	rep := MustFindRepository()

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
