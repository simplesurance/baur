package command

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/simplesurance/baur"
	"github.com/spf13/cobra"
)

var showPrintPathOnly bool

func init() {
	showCmd.Flags().BoolVar(&showPrintPathOnly, "path-only", false, "only print the path of the application")
	rootCmd.AddCommand(showCmd)
}

const showExampleHelp = `
baur show claim-service		        show informations about the claim-service application
baur show --path-only claim-service	show the path of the directory of the claim-service application
baur show .		                show informations about the application in the current directory
baur show		                show informations about the repository
`

var showCmd = &cobra.Command{
	Use:     "show [<APP-NAME>|<PATH>]",
	Short:   "shows informations about applications in the repository",
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
		fmt.Println(app.Dir)
		return
	}

	fmt.Printf("# Application Information\n")
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 8, ' ', 0)

	fmt.Fprintf(tw, "Name:\t%s\n", app.Name)
	fmt.Fprintf(tw, "Directory:\t%s\n", app.Dir)
	fmt.Fprintf(tw, "Build Command:\t%s\n", app.BuildCmd)

	if len(app.Artifacts) == 0 {
		fmt.Fprintf(tw, "Artifacts:-None-\n")
	} else {
		fmt.Fprintf(tw, "Artifacts:\t\n")
		for i, art := range app.Artifacts {
			fmt.Fprintf(tw, "\tLocal:\t%s\n", art.LocalPath())
			fmt.Fprintf(tw, "\tRemote:\t%s\n", art.UploadDestination())
			if i < len(app.Artifacts)-1 {
				fmt.Fprintf(tw, "\t\t\n")
			}
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

	app := mustArgToApp(rep, args[0])

	showApplicationInformation(app)

}
