package command

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur/v3/internal/command/flag"
	"github.com/simplesurance/baur/v3/internal/command/term"
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
	format   *flag.Format
}

func newLsAppsCmd() *lsAppsCmd {
	cmd := lsAppsCmd{
		Command: cobra.Command{
			Use:               "apps [APP_NAME|APP_DIR]...",
			Short:             "list applications",
			Args:              cobra.ArbitraryArgs,
			ValidArgsFunction: completeAppNameAndAppDir,
		},

		format: flag.NewFormatFlag(),
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

	cmd.Flags().Var(cmd.format, "format", cmd.format.Usage(term.Highlight))
	_ = cmd.format.RegisterFlagCompletion(&cmd.Command)

	cmd.Flags().BoolVar(&cmd.csv, "csv", false,
		"List applications in RFC4180 CSV format")
	_ = cmd.Flags().MarkDeprecated("csv", "use --format=csv instead")

	cmd.MarkFlagsMutuallyExclusive("format", "csv")

	cmd.Flags().BoolVarP(&cmd.quiet, "quiet", "q", false,
		"Suppress printing a header and progress dots")

	cmd.Flags().BoolVar(&cmd.absPaths, "abs-path", false,
		"Show absolute instead of relative paths")

	cmd.Flags().VarP(cmd.fields, "fields", "f",
		cmd.fields.Usage(term.Highlight))

	cmd.PreRun = func(*cobra.Command, []string) {
		if cmd.csv {
			cmd.format.Val = flag.FormatCSV
		}
	}

	return &cmd

}

func (c *lsAppsCmd) createHeader() []string {
	headers := make([]string, 0, len(c.fields.Fields))

	if c.format.Val != flag.FormatJSON && (c.format.Val == flag.FormatCSV || c.quiet) {
		return nil
	}

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

	repo := mustFindRepository()
	apps := mustArgToApps(repo, args)

	headers := c.createHeader()

	formatter := mustNewFormatter(c.format.Val, headers)

	baur.SortAppsByName(apps)

	for _, app := range apps {
		row := c.assembleRow(app)
		mustWriteRow(formatter, row...)
	}

	exitOnErr(formatter.Flush())
}

func (c *lsAppsCmd) assembleRow(app *baur.App) []any {
	row := make([]any, 0, len(c.fields.Fields))

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
