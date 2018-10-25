package command

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur/format"
	"github.com/simplesurance/baur/format/csv"
	"github.com/simplesurance/baur/format/table"
	"github.com/simplesurance/baur/log"
	"github.com/simplesurance/baur/storage"
)

type buildsShowConf struct {
	csv bool
}

var buildsShowCmd = &cobra.Command{
	Use:   "show <BUILD-ID>",
	Short: "show informations about a build",
	Run:   buildsShow,
	Args:  cobra.ExactArgs(1),
}

var buildsShowConfig buildsShowConf

func init() {
	buildsShowCmd.Flags().BoolVar(&buildsShowConfig.csv, "csv", false,
		"Show output in RFC4180 CSV format")

	buildsCmd.AddCommand(buildsShowCmd)
}

func buildsShow(cmd *cobra.Command, args []string) {
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

	if buildsShowConfig.csv {
		formatter = csv.New(nil, os.Stdout)
	} else {
		formatter = table.New(nil, os.Stdout)
	}

	mustWriteRow(formatter, []interface{}{fmtVertTitle(buildsShowConfig.csv, "Application"), build.Application.Name})
	mustWriteRow(formatter, []interface{}{fmtVertTitle(buildsShowConfig.csv, "Build ID"), build.ID})

	mustWriteRow(formatter, []interface{}{fmtVertTitle(buildsShowConfig.csv, "Build Started At"), build.StartTimeStamp})
	mustWriteRow(formatter, []interface{}{
		fmtVertTitle(buildsShowConfig.csv, "Build Duration"),
		fmt.Sprintf("%.2f s", build.StopTimeStamp.Sub(build.StartTimeStamp).Seconds()),
	})

	mustWriteRow(formatter, []interface{}{fmtVertTitle(buildsShowConfig.csv, "Git Commit"), vcsStr(&build.VCSState)})

	mustWriteRow(formatter, []interface{}{fmtVertTitle(buildsShowConfig.csv, "Total Input Digest"), build.TotalInputDigest})

	if len(build.Outputs) > 0 {
		mustWriteRow(formatter, []interface{}{})
		mustWriteRow(formatter, []interface{}{fmtVertTitle(buildsShowConfig.csv, "Outputs")})
	}
	for i, o := range build.Outputs {
		mustWriteRow(formatter, []interface{}{fmtVertTitle(buildsShowConfig.csv, "URI"), o.Upload.URI})
		mustWriteRow(formatter, []interface{}{fmtVertTitle(buildsShowConfig.csv, "Digest"), o.Digest})
		mustWriteRow(formatter, []interface{}{
			fmtVertTitle(buildsShowConfig.csv, "Size"),
			fmt.Sprintf("%.2f MiB", float64(o.SizeBytes)/1024/1024),
		})
		mustWriteRow(formatter, []interface{}{
			fmtVertTitle(buildsShowConfig.csv, "Upload Duration"),
			fmt.Sprintf("%.2f s", o.Upload.UploadDuration.Seconds()),
		})
		if i+1 < len(build.Outputs) {
			mustWriteRow(formatter, []interface{}{})
		}
	}

	if err := formatter.Flush(); err != nil {
		log.Fatalln(err)
	}

}
