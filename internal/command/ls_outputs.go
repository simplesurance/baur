package command

import (
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur/v2/internal/command/term"
	"github.com/simplesurance/baur/v2/internal/format"
	"github.com/simplesurance/baur/v2/internal/format/csv"
	"github.com/simplesurance/baur/v2/internal/format/table"
	"github.com/simplesurance/baur/v2/internal/log"
	"github.com/simplesurance/baur/v2/pkg/storage"
)

type lsOutputsCmd struct {
	cobra.Command

	quiet bool
	csv   bool
}

func init() {
	lsCmd.AddCommand(&newLsOutputsCmd().Command)
}

func newLsOutputsCmd() *lsOutputsCmd {
	cmd := lsOutputsCmd{
		Command: cobra.Command{
			Use:   "outputs <RUN_ID>",
			Short: "list outputs of a task run",
			Args:  cobra.ExactArgs(1),
		},
	}

	cmd.Run = cmd.run

	cmd.Flags().BoolVar(&cmd.csv, "csv", false,
		"Show output in RFC4180 CSV format")

	cmd.Flags().BoolVarP(&cmd.quiet, "quiet", "q", false,
		"Only show URIs")

	return &cmd
}

func (c *lsOutputsCmd) run(cmd *cobra.Command, args []string) {
	repo := mustFindRepository()
	pgClient := mustNewCompatibleStorage(repo)
	defer pgClient.Close()

	taskRunID, err := strconv.Atoi(args[0])
	if err != nil {
		stderr.Printf("'%s' is not a numeric task run ID\n", args[0])
		exitFunc(1)
	}

	_, err = pgClient.TaskRun(ctx, taskRunID)
	if err != nil {
		if err == storage.ErrNotExist {
			stderr.Printf("task run with ID %d does not exist", taskRunID)
			exitFunc(1)
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

	formatter := getLsOutputsFormatter(c.quiet, c.csv)

	for _, o := range outputs {
		for _, upload := range o.Uploads {
			if c.quiet {
				mustWriteRow(formatter, upload.URI)
				continue
			}

			mustWriteRow(formatter,
				upload.URI,
				o.Digest,
				term.FormatSize(o.SizeBytes, term.FormatBaseWithoutUnitName(c.csv)),
				term.FormatDuration(
					upload.UploadStopTimestamp.Sub(upload.UploadStartTimestamp),
					term.FormatBaseWithoutUnitName(c.csv),
				),
				o.Type,
				upload.Method,
			)
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
		"Size",
		"Upload Duration (s)",
		"Output Type",
		"Method",
	}

	return table.New(headers, os.Stdout)
}
