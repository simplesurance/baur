package command

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/simplesurance/baur/log"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(ShowCmd)
}

const showExampleHelp = `
baur show claim-service    shows informations about the appliation with the name claim-service
`

var ShowCmd = &cobra.Command{
	Use:     "show <APP-NAME>",
	Short:   "shows informations about applications in the repository",
	Example: strings.TrimSpace(showExampleHelp),
	Run:     show,
	Args:    cobra.ExactArgs(1),
}

func show(cmd *cobra.Command, args []string) {
	rep := mustFindRepository()

	app, err := rep.AppByName(args[0])
	if err != nil {
		if os.IsNotExist(err) {
			log.Fatalf("could not find application with name '%s'\n", args[0])
		}
		log.Fatalln(err)
	}
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 8, ' ', 0)

	fmt.Fprintf(tw, "Name:\t%s\n", app.Name)
	fmt.Fprintf(tw, "Directory:\t%s\n", app.Dir)
	fmt.Fprintf(tw, "Build Command:\t%s\n", app.BuildCmd)
	tw.Flush()
}
