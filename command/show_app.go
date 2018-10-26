package command

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur/format"
	"github.com/simplesurance/baur/format/csv"
	"github.com/simplesurance/baur/format/table"
	"github.com/simplesurance/baur/log"
)

type showAppConf struct {
	csv bool
}

var showAppCmd = &cobra.Command{
	Use:   "app <APP-NAME>|<PATH>",
	Short: "show configuration of an application",
	Run:   showApp,
	Args:  cobra.ExactArgs(1),
}

var showAppConfig showAppConf

func init() {
	showAppCmd.Flags().BoolVar(&showAppConfig.csv, "csv", false,
		"Show output in RFC4180 CSV format")

	showCmd.AddCommand(showAppCmd)
}

func showApp(cmd *cobra.Command, args []string) {
	var formatter format.Formatter

	repo := MustFindRepository()
	app := mustArgToApp(repo, args[0])

	if showAppConfig.csv {
		formatter = csv.New(nil, os.Stdout)
	} else {
		formatter = table.New(nil, os.Stdout)
	}

	mustWriteRow(formatter, []interface{}{fmtVertTitle(showAppConfig.csv, "Name"), app.Name})
	mustWriteRow(formatter, []interface{}{fmtVertTitle(showAppConfig.csv, "Path"), app.RelPath})
	mustWriteRow(formatter, []interface{}{fmtVertTitle(showAppConfig.csv, "Build Command"), app.BuildCmd})

	if len(app.Outputs) != 0 {
		mustWriteRow(formatter, []interface{}{})
		mustWriteRow(formatter, []interface{}{fmtVertTitle(showAppConfig.csv, "Outputs")})

		for i, art := range app.Outputs {
			mustWriteRow(formatter, []interface{}{fmtVertTitle(showAppConfig.csv, "Type"), art.Type()})
			mustWriteRow(formatter, []interface{}{fmtVertTitle(showAppConfig.csv, "Local"), art.String()})
			mustWriteRow(formatter, []interface{}{fmtVertTitle(showAppConfig.csv, "Remote"), art.UploadDestination()})

			if i+1 < len(app.Outputs) {
				mustWriteRow(formatter, []interface{}{})
			}
		}
	}

	if len(app.BuildInputPaths) != 0 {
		mustWriteRow(formatter, []interface{}{})
		mustWriteRow(formatter, []interface{}{fmtVertTitle(showAppConfig.csv, "Inputs")})

		for _, bi := range app.BuildInputPaths {
			mustWriteRow(formatter, []interface{}{fmtVertTitle(showAppConfig.csv, bi.Type()), bi.String()})
		}

	}

	if err := formatter.Flush(); err != nil {
		log.Fatalln(err)
	}
}
