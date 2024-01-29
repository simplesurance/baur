package command

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur/v3/internal/command/flag"
	"github.com/simplesurance/baur/v3/internal/command/term"
	"github.com/simplesurance/baur/v3/internal/format/json"
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
	var rows []*appRow

	repo := mustFindRepository()
	apps := mustArgToApps(repo, args)

	if !c.quiet && c.format.Val == flag.FormatPlain {
		headers = c.createHeader()
	}

	formatter := mustNewFormatter(c.format.Val, headers)

	baur.SortAppsByName(apps)

	for _, app := range apps {
		row := c.assembleRow(app)

		if c.format.Val == flag.FormatJSON {
			rows = append(rows, row)
		} else {
			mustWriteRow(formatter, row.asOrderedSlice(c.fields.Fields)...)
		}
	}

	if c.format.Val == flag.FormatJSON {
		exitOnErr(json.Encode(stdout, rows, c.fields.Fields))
		return
	}

	exitOnErr(formatter.Flush())
}

func (c *lsAppsCmd) assembleRow(app *baur.App) *appRow {
	var row appRow

	for _, f := range c.fields.Fields {
		switch f {
		case lsAppNameParam:
			row.Name = &app.Name

		case lsAppPathParam:
			if c.absPaths {
				row.Path = &app.Path
			} else {
				row.Path = &app.RelPath
			}
		}
	}

	return &row
}

type appRow struct {
	Name *string
	Path *string
}

func (r *appRow) asOrderedSlice(order []string) []any {
	result := make([]any, 0, len(order))
	for _, f := range order {
		switch f {
		case lsAppNameParam:
			result = sliceAppendNilAsEmpty(result, r.Name)
		case lsAppPathParam:
			result = sliceAppendNilAsEmpty(result, r.Path)
		default:
			panic(fmt.Sprintf("BUG: asOrderedSlice: got unsupported field name %q in order list", f))
		}
	}

	return result
}

func (r *appRow) AsMap(fields []string) map[string]any {
	m := make(map[string]any, len(fields))
	for _, f := range fields {
		switch f {
		case lsAppNameParam:
			m["AppName"] = r.Name
		case lsAppPathParam:
			m["Path"] = r.Path
		default:
			panic(fmt.Sprintf("BUG: asMap: got unsupported field name %q in order list", f))
		}
	}

	return m
}
