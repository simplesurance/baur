package command

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur/v3/internal/command/flag"
	"github.com/simplesurance/baur/v3/internal/command/term"
	"github.com/simplesurance/baur/v3/internal/format"
	"github.com/simplesurance/baur/v3/internal/format/csv"
	"github.com/simplesurance/baur/v3/internal/format/table"
	"github.com/simplesurance/baur/v3/pkg/baur"
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
			Use:               "apps [APP_NAME|APP_DIR]...",
			Short:             "list applications",
			Args:              cobra.ArbitraryArgs,
			ValidArgsFunction: completeAppNameAndAppDir,
		},

		fields: flag.MustNewFields(
			[]string{
				lsAppNameParam,
				lsAppPathParam,
			},
			[]string{
				lsAppNameParam,
				lsAppPathParam,
			},
		),
	}

	cmd.Run = cmd.run

	cmd.Flags().BoolVar(&cmd.csv, "csv", false,
		"List applications in RFC4180 CSV format")

	cmd.Flags().BoolVarP(&cmd.quiet, "quiet", "q", false,
		"Suppress printing a header and progress dots")

	cmd.Flags().BoolVar(&cmd.absPaths, "abs-path", false,
		"Show absolute instead of relative paths")

	cmd.Flags().VarP(cmd.fields, "fields", "f",
		cmd.fields.Usage(term.Highlight))

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

func (c *lsAppsCmd) run(_ *cobra.Command, args []string) {
	var headers []string
	var formatter format.Formatter

	repo := mustFindRepository()
	apps := mustArgToApps(repo, args)

	if !c.quiet && !c.csv {
		headers = c.createHeader()
	}

	if c.csv {
		formatter = csv.New(headers, stdout)
	} else {
		formatter = table.New(headers, stdout)
	}

	baur.SortAppsByName(apps)

	for _, app := range apps {
		row := c.assembleRow(app)

		mustWriteRow(formatter, row...)
	}

	err := formatter.Flush()
	exitOnErr(err)
}

func (c *lsAppsCmd) assembleRow(app *baur.App) []any {
	row := make([]any, 0, 2)

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
