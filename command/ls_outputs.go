package command

import (
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur/format"
	"github.com/simplesurance/baur/format/csv"
	"github.com/simplesurance/baur/format/table"
	"github.com/simplesurance/baur/log"
)

var lsOutputsCmd = &cobra.Command{
	Use:   "outputs <BUILD-ID>",
	Short: "list outputs for a build",
	Run:   lsOutputs,
	Args:  cobra.ExactArgs(1),
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
		log.Fatalf("'%s' is not a numeric build ID", args[0])
	}

	exist, err := pgClient.BuildExist(buildID)
	if err != nil {
		log.Fatalln(err)
	}

	if !exist {
		log.Fatalf("build with ID %d does not exist", buildID)
	}

	outputs, err := pgClient.GetBuildOutputs(buildID)
	if err != nil {
		log.Fatalln(err)
	}

	formatter := getLsOutputsFormatter(lsOutputsConf.quiet, lsOutputsConf.csv)

	for _, o := range outputs {
		var row []interface{}

		if lsOutputsConf.quiet {
			row = []interface{}{o.Upload.URI}
		} else {
			row = []interface{}{
				o.Type,
				o.Upload.URI,
				o.Digest,
				bytesToMib(int(o.SizeBytes)),
				o.Upload.UploadDuration,
			}
		}

		mustWriteRow(formatter, row)
	}

	if err = formatter.Flush(); err != nil {
		log.Fatalln(err)
	}
}

func getLsOutputsFormatter(isQuiet, isCsv bool) format.Formatter {
	var headers []string

	if isCsv {
		return csv.New(headers, os.Stdout)
	}

	if isQuiet {
		return table.New(headers, os.Stdout)
	}

	headers = []string{
		"Type",
		"URI",
		"Digest",
		"Size (MiB)",
		"Upload Duration",
	}

	return table.New(headers, os.Stdout)
}
