package command

import (
	"errors"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur/v5/internal/command/flag"
	"github.com/simplesurance/baur/v5/internal/command/term"
	"github.com/simplesurance/baur/v5/internal/log"
	"github.com/simplesurance/baur/v5/pkg/storage"
)

type lsOutputsCmd struct {
	cobra.Command

	quiet  bool
	format *flag.OneOf
}

func init() {
	lsCmd.AddCommand(&newLsOutputsCmd().Command)
}

func newLsOutputsCmd() *lsOutputsCmd {
	cmd := lsOutputsCmd{
		Command: cobra.Command{
			Use:               "outputs <RUN_ID>",
			Short:             "list outputs of a task run",
			Args:              cobra.ExactArgs(1),
			ValidArgsFunction: cobra.NoFileCompletions,
		},
		format: flag.NewFormatFlag(),
	}

	cmd.Run = cmd.run

	cmd.Flags().Var(cmd.format, flag.FormatFlagName, cmd.format.Usage(term.Highlight))
	_ = cmd.format.RegisterFlagCompletion(&cmd.Command)

	cmd.Flags().BoolVarP(&cmd.quiet, "quiet", "q", false,
		"only show the URIs of the outputs in plain and csv format")

	return &cmd
}

func (c *lsOutputsCmd) run(_ *cobra.Command, args []string) {
	taskRunID, err := strconv.Atoi(args[0])
	if err != nil {
		stderr.Printf("'%s' is not a numeric task run ID\n", args[0])
		exitFunc(exitCodeError)
	}

	repo := mustFindRepository()
	pgClient := mustNewCompatibleStorageRepo(repo)
	defer pgClient.Close()

	_, err = pgClient.TaskRun(ctx, taskRunID)
	if err != nil {
		if errors.Is(err, storage.ErrNotExist) {
			stderr.Printf("task run with ID %d does not exist", taskRunID)
			exitFunc(exitCodeError)
		}
	}

	outputs, err := pgClient.Outputs(ctx, taskRunID)
	if err != nil {
		if errors.Is(err, storage.ErrNotExist) {
			log.Debugf("task run with ID %d has no outputs", taskRunID)
		} else {
			exitOnErr(err)
		}
	}

	headers := c.createHeader()
	formatter := mustNewFormatter(c.format.Val, headers)

	withoutUnits := c.format.Val != flag.FormatPlain
	for _, o := range outputs {
		for _, upload := range o.Uploads {
			var bytes any
			var duration any

			if c.quiet {
				mustWriteRow(formatter, upload.URI)
				continue
			}

			if c.format.Val == flag.FormatJSON {
				bytes = o.SizeBytes
				duration = upload.UploadStopTimestamp.Sub(upload.UploadStartTimestamp).Seconds()
			} else {
				bytes = term.FormatSize(
					o.SizeBytes,
					term.FormatBaseWithoutUnitName(withoutUnits),
				)
				duration = term.FormatDuration(
					upload.UploadStopTimestamp.Sub(upload.UploadStartTimestamp),
					term.FormatBaseWithoutUnitName(withoutUnits),
				)
			}

			mustWriteRow(formatter,
				upload.URI,
				o.Digest,
				bytes,
				duration,
				o.Type,
				upload.Method,
			)
		}
	}

	err = formatter.Flush()
	exitOnErr(err)
}

func (c *lsOutputsCmd) createHeader() []string {
	if c.format.Val == flag.FormatJSON {
		return []string{
			"URI",
			"Digest",
			"Bytes",
			"UploadDurationSeconds",
			"OutputType",
			"UploadMethod",
		}
	}

	if c.quiet {
		return nil
	}

	return []string{
		"URI",
		"Digest",
		"Size",
		"Upload Duration (s)",
		"Output Type",
		"Method",
	}
}
