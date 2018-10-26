package command

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur/format"
	"github.com/simplesurance/baur/format/table"
	"github.com/simplesurance/baur/log"
	"github.com/simplesurance/baur/storage"
)

var showBuildCmd = &cobra.Command{
	Use:   "build <BUILD-ID>",
	Short: "show information about a build",
	Run:   showBuild,
	Args:  cobra.ExactArgs(1),
}

func init() {
	showCmd.AddCommand(showBuildCmd)
}

func showBuild(cmd *cobra.Command, args []string) {
	var formatter format.Formatter

	buildID, err := strconv.Atoi(args[0])
	if err != nil {
		log.Fatalf("%q is not a numeric build ID\n", args[0])
	}

	repo := MustFindRepository()
	storageClt := MustGetPostgresClt(repo)

	build, err := storageClt.GetBuildWithoutInputs(int(buildID))
	if err != nil {
		if err == storage.ErrNotExist {
			log.Fatalf("build with id %d does not exist\n", buildID)
		}

		log.Fatalln(err)
	}

	formatter = table.New(nil, os.Stdout)

	mustWriteRow(formatter, []interface{}{underline("General:")})
	mustWriteRow(formatter, []interface{}{"", "Name:", highlight(build.Application.Name)})
	mustWriteRow(formatter, []interface{}{"", "ID:", highlight(build.ID)})

	mustWriteRow(formatter, []interface{}{"", "Started At:", highlight(build.StartTimeStamp)})
	mustWriteRow(formatter, []interface{}{
		"",
		"Duration:",
		highlight(fmt.Sprintf("%.2f s", build.StopTimeStamp.Sub(build.StartTimeStamp).Seconds())),
	})

	mustWriteRow(formatter, []interface{}{"", "Git Commit:", highlight(vcsStr(&build.VCSState))})

	mustWriteRow(formatter, []interface{}{"", "Total Input Digest:", highlight(build.TotalInputDigest)})

	if len(build.Outputs) > 0 {
		mustWriteRow(formatter, []interface{}{})
		mustWriteRow(formatter, []interface{}{underline("Outputs:")})
	}
	for i, o := range build.Outputs {
		mustWriteRow(formatter, []interface{}{"", "Type:", highlight(o.Type)})
		mustWriteRow(formatter, []interface{}{"", "URI:", highlight(o.Upload.URI)})
		mustWriteRow(formatter, []interface{}{"", "Digest:", highlight(o.Digest)})
		mustWriteRow(formatter, []interface{}{
			"",
			"Size:",
			highlight(fmt.Sprintf("%.2f MiB", float64(o.SizeBytes)/1024/1024)),
		})
		mustWriteRow(formatter, []interface{}{
			"",
			"Upload Duration:",
			highlight(fmt.Sprintf("%.2f s", o.Upload.UploadDuration.Seconds())),
		})
		if i+1 < len(build.Outputs) {
			mustWriteRow(formatter, []interface{}{})
		}
	}

	if err := formatter.Flush(); err != nil {
		log.Fatalln(err)
	}

}
