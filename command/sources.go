package command

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/simplesurance/baur"
	"github.com/simplesurance/baur/log"
	"github.com/spf13/cobra"
)

var sourcesShowDigest bool

func init() {
	sourcesCmd.Flags().BoolVar(&sourcesShowDigest, "digest", false,
		"show file digests")
	rootCmd.AddCommand(sourcesCmd)
}

const sourcesLongHelp = `
Shows source file paths of an application.

The paths to the source files must be configured in the .app.toml file.
`

var sourcesCmd = &cobra.Command{
	Use:   "sources [<APP-NAME>|<PATH>]",
	Args:  cobra.ExactArgs(1),
	Short: "list source files of an application",
	Long:  strings.TrimSpace(sourcesLongHelp),
	Run:   sources,
}

func sources(cmd *cobra.Command, args []string) {
	var alLFiles []*baur.File

	rep := mustFindRepository()
	app := mustArgToApp(rep, args[0])

	if len(app.Sources) == 0 {
		log.Fatalf("No source files have been configured in the %s file of %s\n", baur.AppCfgFile, app.Name)
	}

	for _, s := range app.Sources {
		paths, err := s.Resolve()
		if err != nil {
			log.Fatalln("resolving source paths failed:", err)
		}

		alLFiles = append(alLFiles, paths...)
	}

	sort.Slice(alLFiles, func(i, j int) bool {
		return alLFiles[i].RelPath() < alLFiles[j].RelPath()
	})

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, p := range alLFiles {
		if !sourcesShowDigest {
			fmt.Fprintf(tw, "%s\n", p.Path())
			continue
		}

		d, err := p.Digest()
		if err != nil {
			log.Fatalln("creating digest failed:", err)
		}

		fmt.Fprintf(tw, "%s\t%s\n", p.Path(), d)
	}

	tw.Flush()
}
