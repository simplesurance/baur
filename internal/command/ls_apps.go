package command

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur"
	"github.com/simplesurance/baur/format"
	"github.com/simplesurance/baur/format/csv"
	"github.com/simplesurance/baur/format/table"
	"github.com/simplesurance/baur/internal/command/flag"
	"github.com/simplesurance/baur/internal/command/terminal"
)

const (
	lsAppNameHeader = "Name"
	lsAppNameParam  = "name"
	lsAppPathHeader = "Path"
	lsAppPathParam  = "path"
)

func init() {
	lsCmd.AddCommand(&newLsAppsCmd().Command)
}

type lsAppsCmd struct {
	cobra.Command

	csv      bool
	quiet    bool
	absPaths bool
	fields   *flag.Fields
}

func newLsAppsCmd() *lsAppsCmd {
	cmd := lsAppsCmd{
		Command: cobra.Command{
			Use:   "apps [<APP-NAME>|<PATH>]...",
			Short: "list applications",
			Args:  cobra.ArbitraryArgs,
		},

		fields: flag.NewFields([]string{
			lsAppNameParam,
			lsAppPathParam,
		}),
	}

	cmd.Run = cmd.run

	cmd.Flags().BoolVar(&cmd.csv, "csv", false,
		"List applications in RFC4180 CSV format")

	cmd.Flags().BoolVarP(&cmd.quiet, "quiet", "q", false,
		"Suppress printing a header and progress dots")

	cmd.Flags().BoolVar(&cmd.absPaths, "abs-path", false,
		"Show absolute instead of relative paths")

	cmd.Flags().VarP(cmd.fields, "fields", "f",
		cmd.fields.Usage(terminal.Highlight))

	return &cmd

}

func (c *lsAppsCmd) createHeader() []string {
	var headers []string

	for _, f := range c.fields.Fields {
		switch f {
		case lsAppNameParam:
			headers = append(headers, lsAppNameHeader)
		case lsAppPathParam:
			headers = append(headers, lsAppPathHeader)

		default:
			panic(fmt.Sprintf("unsupported value '%v' in fields parameter", f))
		}
	}

	return headers
}

func (c *lsAppsCmd) run(cmd *cobra.Command, args []string) {
	var headers []string
	var formatter format.Formatter

	repo := MustFindRepository()
	apps := mustArgToApps(repo, args)

	if !c.quiet && !c.csv {
		headers = c.createHeader()
	}

	if c.csv {
		formatter = csv.New(headers, os.Stdout)
	} else {
		formatter = table.New(headers, os.Stdout)
	}

	baur.SortAppsByName(apps)

	for _, app := range apps {
		row := c.assembleRow(app)

		err := formatter.WriteRow(row)
		exitOnErr(err)
	}

	err := formatter.Flush()
	exitOnErr(err)
}

func (c *lsAppsCmd) assembleRow(app *baur.App) []interface{} {
	row := make([]interface{}, 0, 2)

	for _, f := range c.fields.Fields {
		switch f {
		case lsAppNameParam:
			row = append(row, app.Name)

		case lsAppPathParam:
			if c.absPaths {
				row = append(row, app.Path)
			} else {
				row = append(row, app.RelPath)
			}
		}
	}

	return row
}
