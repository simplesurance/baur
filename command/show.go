package command

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/simplesurance/baur/log"
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
baur show		                show repository root
`

var showCmd = &cobra.Command{
	Use:     "show [<APP-NAME>]",
	Short:   "shows informations about applications in the repository",
	Example: strings.TrimSpace(showExampleHelp),
	Run:     show,
	Args:    cobra.MaximumNArgs(1),
}

func show(cmd *cobra.Command, args []string) {
	rep := mustFindRepository()

	if len(args) == 0 {
		fmt.Println(rep.Path)
		os.Exit(0)
	}
	app, err := rep.AppByName(args[0])
	if err != nil {
		if os.IsNotExist(err) {
			log.Fatalf("could not find application with name '%s'\n", args[0])
		}
		log.Fatalln(err)
	}

	if showPrintPathOnly {
		fmt.Println(app.Dir)
		os.Exit(0)
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 8, ' ', 0)

	fmt.Fprintf(tw, "Name:\t%s\n", app.Name)
	fmt.Fprintf(tw, "Directory:\t%s\n", app.Dir)
	fmt.Fprintf(tw, "Build Command:\t%s\n", app.BuildCmd)
	tw.Flush()
}
