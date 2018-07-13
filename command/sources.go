package command

import (
	"fmt"
	"sort"
	"strings"

	"github.com/simplesurance/baur"
	"github.com/simplesurance/baur/log"
	"github.com/spf13/cobra"
)

func init() {
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
	var allPaths []string

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
		allPaths = append(allPaths, paths...)
	}

	if len(allPaths) == 0 {
		log.Fatalln("configured source file paths resolved to 0 files, ensure the configuration is correct")
	}

	sort.Strings(allPaths)

	for _, p := range allPaths {
		fmt.Println(p)
	}
}
