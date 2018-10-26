package command

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur/format"
	"github.com/simplesurance/baur/format/table"
	"github.com/simplesurance/baur/log"
)

var showAppCmd = &cobra.Command{
	Use:   "app <APP-NAME>|<PATH>",
	Short: "show configuration of an application",
	Run:   showApp,
	Args:  cobra.ExactArgs(1),
}

func init() {
	showCmd.AddCommand(showAppCmd)
}

func showApp(cmd *cobra.Command, args []string) {
	var formatter format.Formatter

	repo := MustFindRepository()
	app := mustArgToApp(repo, args[0])

	formatter = table.New(nil, os.Stdout)

	mustWriteRow(formatter, []interface{}{underline("General:")})
	mustWriteRow(formatter, []interface{}{"", "Name:", highlight(app.Name)})
	mustWriteRow(formatter, []interface{}{"", "Path:", highlight(app.RelPath)})
	mustWriteRow(formatter, []interface{}{"", "Build Command:", highlight(app.BuildCmd)})

	if len(app.Outputs) != 0 {
		mustWriteRow(formatter, []interface{}{})
		mustWriteRow(formatter, []interface{}{underline("Outputs:")})

		for i, art := range app.Outputs {
			mustWriteRow(formatter, []interface{}{"", "Type:", highlight(art.Type())})
			mustWriteRow(formatter, []interface{}{"", "Local:", highlight(art.String())})
			mustWriteRow(formatter, []interface{}{"", "Remote:", highlight(art.UploadDestination())})

			if i+1 < len(app.Outputs) {
				mustWriteRow(formatter, []interface{}{})
			}
		}
	}

	if len(app.BuildInputPaths) != 0 {
		mustWriteRow(formatter, []interface{}{})
		mustWriteRow(formatter, []interface{}{underline("Inputs:")})

		for i, bi := range app.BuildInputPaths {
			mustWriteRow(formatter, []interface{}{"", "Type:", highlight(bi.Type())})
			mustWriteRow(formatter, []interface{}{"", "Configuration:", highlight(bi.String())})

			if i+1 < len(app.BuildInputPaths) {
				mustWriteRow(formatter, []interface{}{})
			}
		}

	}

	if err := formatter.Flush(); err != nil {
		log.Fatalln(err)
	}
}
