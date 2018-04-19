package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/simplesurance/baur/app"
	"github.com/simplesurance/baur/cfg"
	"github.com/simplesurance/baur/discover"
	"github.com/simplesurance/baur/sblog"
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
	ctx := mustInitCtx()

	dirs, err := discover.ApplicationDirs(ctx.RepositoryCfg.Discover.Dirs,
		cfg.ApplicationFile, ctx.RepositoryCfg.Discover.SearchDepth)
	if err != nil {
		sblog.Fatal("discovering applications failed: ", err)
	}

	if len(dirs) == 0 {
		sblog.Fatalf("could not find any applications\n"+
			"- ensure the [Discover] is correct in %s\n"+
			"- ensure that you have >1 application dirs "+
			"containing a %s file",
			ctx.RepositoryCfgPath, cfg.ApplicationFile)
	}

	apps := make([]*app.App, 0, len(dirs))
	for _, d := range dirs {
		a, err := app.New(d)
		if err != nil {
			sblog.Fatalf("could not get application informations: %s", err)
		}
		apps = append(apps, a)
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 8, ' ', 0)
	fmt.Fprintf(tw, "# Name\tDirectory\n")
	for _, a := range apps {
		fmt.Fprintf(tw, "%s\t%s\n", a.Name, a.Dir)
	}
	tw.Flush()
}
