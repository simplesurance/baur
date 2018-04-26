package command

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/simplesurance/baur"
	"github.com/simplesurance/baur/log"
	"github.com/simplesurance/baur/term"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(lsCmd)
}

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "list all applications in the repository",
	Run:   ls,
}

func ls(cmd *cobra.Command, args []string) {
	rep := mustFindRepository()

	apps, err := rep.FindApps()
	if err != nil {
		log.Fatalln(err)
	}

	if len(apps) == 0 {
		log.Fatalf("could not find any applications\n"+
			"- ensure the [Discover] section is correct in %s\n"+
			"- ensure that you have >1 application dirs "+
			"containing a %s file",
			rep.CfgPath, baur.AppCfgFile)
	}

	baur.SortAppsByName(apps)

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 8, ' ', 0)
	fmt.Fprintf(tw, "# Name\tDirectory\n")
	for _, a := range apps {
		fmt.Fprintf(tw, "%s\t%s\n", a.Name, a.Dir)
	}
	tw.Flush()

	term.PrintSep()
	fmt.Printf("Total: %v\n", len(apps))
}
