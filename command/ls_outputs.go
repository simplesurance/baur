package command

import (
	"os"
	"strconv"

	"github.com/simplesurance/baur/format"
	"github.com/simplesurance/baur/format/csv"
	"github.com/simplesurance/baur/format/table"
	"github.com/simplesurance/baur/log"
	"github.com/spf13/cobra"
)

var lsOutputsCmd = &cobra.Command{
	Use:     "outputs <BUILD-ID>",
	Short:   "lists outputs for a build",
	Example: "baur ls outputs 13",
	Run:     lsOutputs,
	Args:    cobra.ExactArgs(1),
}

type lsOutputsConfig struct {
	quiet bool
	csv   bool
}

var lsOutputsConf lsOutputsConfig

func init() {
	lsOutputsCmd.Flags().BoolVar(&lsOutputsConf.csv, "csv", false,
		"Show output in RFC4180 CSV format")

	lsOutputsCmd.Flags().BoolVarP(&lsOutputsConf.quiet, "quiet", "q", false,
		"Only show URIs")

	lsCmd.AddCommand(lsOutputsCmd)
}

func lsOutputs(cmd *cobra.Command, args []string) {
	repo := MustFindRepository()
	pgClient := MustGetPostgresClt(repo)

	buildID, err := strconv.Atoi(args[0])
	if err != nil {
		log.Fatalln("First arg has to be the build ID")
	} else if _, err2 := pgClient.GetBuild(buildID); err2 != nil {
		log.Fatalf("Build %d doesn't exist", buildID)
	}

	formatter := getLsOutputsFormatter(lsOutputsConf.quiet, lsOutputsConf.csv)

	outputs, err := pgClient.GetBuildOutputs(buildID)
	if err != nil {
		log.Fatalln(err)
	}

	for _, o := range outputs {
		var row []interface{}
		if lsOutputsConf.quiet {
			row = []interface{}{o.Upload.URI}
		} else {
			row = []interface{}{
				o.Type,
				o.Upload.URI,
				o.Digest,
				o.SizeBytes,
				o.Upload.UploadDuration,
			}
		}

		formatter.WriteRow(row)
	}
	formatter.Flush()
}

func getLsOutputsFormatter(isQuiet, isCsv bool) format.Formatter {
	var headers []string
	if !isQuiet && !isCsv {
		headers = []string{
			"Type",
			"URI",
			"Digest",
			"Size",
			"Upload duration",
		}
	}
	if isCsv {
		return csv.New(headers, os.Stdout)
	}

	return table.New(headers, os.Stdout)
}
