package command

import (
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur/format"
	"github.com/simplesurance/baur/format/csv"
	"github.com/simplesurance/baur/format/table"
	"github.com/simplesurance/baur/internal/command/terminal"
	"github.com/simplesurance/baur/log"
	"github.com/simplesurance/baur/storage"
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
	pgClient := mustNewCompatibleStorage(repo)

	taskRunID, err := strconv.Atoi(args[0])
	if err != nil {
		log.Fatalf("'%s' is not a numeric task run ID", args[0])
	}

	_, err = pgClient.TaskRun(ctx, taskRunID)
	if err != nil {
		if err == storage.ErrNotExist {
			log.Fatalf("task run with ID %d does not exist", taskRunID)
		}

	}

	outputs, err := pgClient.Outputs(ctx, taskRunID)
	if err != nil {
		if err == storage.ErrNotExist {
			log.Debugf("task run with ID %d has no outputs", taskRunID)
		} else {
			exitOnErr(err)
		}
	}

	formatter := getLsOutputsFormatter(lsOutputsConf.quiet, lsOutputsConf.csv)

	for _, o := range outputs {
		for _, upload := range o.Uploads {
			var row []interface{}

			if lsOutputsConf.quiet {
				row = []interface{}{upload.URI}
			} else {
				row = []interface{}{
					upload.URI,
					o.Digest,
					terminal.BytesToMib(o.SizeBytes),
					terminal.DurationToStrSeconds(upload.UploadStopTimestamp.Sub(upload.UploadStartTimestamp)),
					o.Type,
					upload.Method,
				}
			}

			mustWriteRow(formatter, row)
		}
	}

	err = formatter.Flush()
	exitOnErr(err)
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
		"URI",
		"Digest",
		"Size (MiB)",
		"Upload Duration (s)",
		"Output Type",
		"Method",
	}

	return table.New(headers, os.Stdout)
}
