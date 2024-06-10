package command

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur/v3/internal/command/term"
	"github.com/simplesurance/baur/v3/internal/log"
	"github.com/simplesurance/baur/v3/internal/output/s3"
	"github.com/simplesurance/baur/v3/pkg/baur"
	"github.com/simplesurance/baur/v3/pkg/storage"
)

var releaseDownloadLongHelp = fmt.Sprintf(`
Download outputs belonging to a release.

The command downloads one instance of each uploaded output of task runs that
belong to the release.
Only outputs that have been uploaded to S3 are downloaded, others are ignored.
The downloaded outputs are stored at the path: %s.

The command can be run without access to the baur repository by specifying the
PostgreSQL URI via the environment variable %s.

When task IDS are specified via %s, the command fails if any of them is not
part of the release or does not have >1 output uploaded to S3.


Exit Codes:
  %d - Success
  %d - Error
  %d - Release does not exist
`,
	term.Highlight("DEST-DIR/TASK-ID/OUTPUT-NAME"),
	term.Highlight(envVarPSQLURL),
	term.Highlight("--tasks"),
	exitCodeSuccess,
	exitCodeError,
	exitCodeNotExist,
)

type releaseDownloadCmd struct {
	cobra.Command
	taskIDs []string
}

func init() {
	releaseCmd.AddCommand(&newReleaseDownloadCmd().Command)
}

func newReleaseDownloadCmd() *releaseDownloadCmd {
	cmd := releaseDownloadCmd{
		Command: cobra.Command{
			Use:   "download RELEASE-NAME DEST-DIR",
			Short: "download outputs of a release",
			Long:  strings.TrimSpace(releaseDownloadLongHelp),
			Args:  cobra.ExactArgs(2),
			ValidArgsFunction: func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
				if len(args) > 0 {
					return nil, cobra.ShellCompDirectiveFilterDirs
				}
				return nil, cobra.ShellCompDirectiveNoFileComp
			},
		},
	}

	cmd.Flags().StringSliceVar(&cmd.taskIDs, "tasks", nil,
		"comma-separated list of Task IDs (APP-NAME.TASK-NAME) for which the outputs are downloaded\n"+
			"(default: all)",
	)

	cmd.Run = cmd.run
	return &cmd
}

func (r *releaseDownloadCmd) run(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()
	releaseName := args[0]
	destDir := args[1]

	psqlURL, err := postgresqlURL()
	exitOnErr(err)

	storageClt := mustNewCompatibleStorage(psqlURL)

	s3clt, err := s3.NewClient(ctx, log.StdLogger)
	exitOnErr(err)

	mgr := baur.NewReleaseManager(&storageClt, s3clt, log.StdLogger)
	downloadCount := 0
	err = mgr.DownloadOutputs(ctx, baur.DownloadOutputsParams{
		ReleaseName: releaseName,
		DestDir:     destDir,
		TaskIDs:     r.taskIDs,
		DownloadStartFn: func(taskID, url, destfilepath string) {
			stdout.Printf(
				"%s: downloading %s -> %s\n",
				term.Highlight(taskID), url, destfilepath,
			)
			downloadCount++
		},
	})

	if errors.Is(err, storage.ErrNotExist) {
		stderr.Printf(
			"release %s does not exist\n",
			term.Highlight(args[0]),
		)
		exitFunc(exitCodeNotExist)
	}

	exitOnErr(err)

	stdout.Printf(
		"\ndownloaded %s outputs %s to %s\n",
		term.Highlight(downloadCount), term.GreenHighlight("successfully"), term.Highlight(destDir),
	)
}
