package command

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/simplesurance/baur/v5/internal/command/flag"
	"github.com/simplesurance/baur/v5/internal/command/term"
	"github.com/simplesurance/baur/v5/pkg/storage"

	"github.com/spf13/cobra"
)

type cleanupDbCmd struct {
	cobra.Command
	before  flag.DateTimeFlagValue
	force   bool
	pretend bool
}

const (
	timeFormat         = "02 Jan 06 15:04:05 MST"
	cleanupDbGracetime = time.Second * 5
)

const (
	deleteTargetTaskRuns = "taskruns"
	deleteTargetReleases = "releases"
)

var cleanupDbLongHelp = fmt.Sprintf(`
Delete old data from the baur database.
The command deletes information about releases or tasks runs that have been
created before a given date from the database.
It also removes records that became dangling because all task runs referencing
them were deleted.
Task runs that are referenced by a release are not deleted.

The command can be run without access to the baur repository by specifying the
PostgreSQL URI via the environment variable %s.
`,
	term.Highlight(envVarPSQLURL),
)

const cleanupDbCmdExample = `
baur cleanup db --force --before=2023.06.01-13:30 releases
baur cleanup db --pretend --before=2023.06.01-13:30 taskruns
`

func init() {
	cleanupCmd.AddCommand(&newCleanupDbCmd().Command)
}

func newCleanupDbCmd() *cleanupDbCmd {
	cmd := cleanupDbCmd{
		Command: cobra.Command{
			Args:      cobra.ExactArgs(1),
			Use:       "db --before=DATETIME releases|taskruns",
			Long:      strings.TrimSpace(cleanupDbLongHelp),
			Example:   strings.TrimSpace(cleanupDbCmdExample),
			ValidArgs: []string{"releases", "taskruns"},
		},
	}

	cmd.Flags().Var(&cmd.before, "before",
		fmt.Sprintf(
			"delete records that have been created before DATETIME\nFormat: %s",
			term.Highlight(flag.DateTimeFormatDescr),
		),
	)
	cmd.Flags().BoolVarP(&cmd.pretend, "pretend", "p", false,
		"do not delete anything, only pretend how many records would be deleted",
	)

	cmd.Flags().BoolVarP(&cmd.force, "force", "f", false,
		fmt.Sprintf(
			"do not wait %s seconds before starting deletion, delete immediately",
			cleanupDbGracetime,
		),
	)

	if err := cmd.MarkFlagRequired("before"); err != nil {
		panic(err)
	}

	cmd.Run = cmd.run

	return &cmd
}

func (c *cleanupDbCmd) run(cmd *cobra.Command, args []string) {
	var op string
	var target string
	var subject string

	psqlURL, err := postgresqlURL()
	exitOnErr(err)

	storageClt := mustNewCompatibleStorage(psqlURL)

	switch args[0] {
	case deleteTargetTaskRuns:
		target = deleteTargetTaskRuns
		subject = "task runs"
	case deleteTargetReleases:
		target = deleteTargetReleases
		subject = "releases"
	default:
		fatalf("internal error: impossible cmdline arguments: %v\n", args)
	}

	if c.pretend {
		op = term.Highlight("pretending to delete")
	} else {
		op = term.Highlight("deleting")
	}

	stdout.Printf(
		"%s %s older then %s",
		op, subject,
		term.Highlight(c.before.Format(timeFormat)),
	)
	if target == deleteTargetTaskRuns {
		stdout.Printf(" and dangling records,\ntasks runs referenced by releases are kept\n")
	} else {
		stdout.Println()
	}

	if !c.force {
		stdout.Printf("starting in %s seconds, press %s to abort\n",
			term.Highlight(cleanupDbGracetime), term.Highlight("CTRL+C"))
		time.Sleep(cleanupDbGracetime)
		stdout.Println("starting deleting...")
	}

	if target == deleteTargetReleases {
		exitOnErr(c.deleteReleases(cmd.Context(), storageClt))
		return
	}

	if target == deleteTargetTaskRuns {
		exitOnErr(c.deleteTaskRuns(cmd.Context(), storageClt))
		return
	}
}

func (c *cleanupDbCmd) deleteReleases(ctx context.Context, storageClt storage.Storer) error {
	startTime := time.Now()
	result, err := storageClt.ReleasesDelete(ctx, c.before.Time, c.pretend)
	if err != nil {
		return err
	}

	stdout.Printf(
		"\n"+
			"deletion %s in %s, deleted records:\n"+
			"%-16s %s\n",
		term.GreenHighlight("successful"),
		term.FormatDuration(time.Since(startTime)),
		"Releases:", term.Highlight(result.DeletedReleases),
	)

	return nil
}

func (c *cleanupDbCmd) deleteTaskRuns(ctx context.Context, storageClt storage.Storer) error {
	var stateStr string

	startTime := time.Now()
	result, err := storageClt.TaskRunsDelete(ctx, c.before.Time, c.pretend)
	if err != nil && result == nil {
		return err
	}

	if err == nil {
		stateStr = term.GreenHighlight("successful")
	} else {
		stateStr = term.RedHighlight("failed")
	}

	stdout.Printf(
		"\n"+
			"deletion %s in %s, deleted records:\n"+
			"%-16s %s\n"+
			"%-16s %s\n"+
			"%-16s %s\n"+
			"%-16s %s\n"+
			"%-16s %s\n"+
			"%-16s %s\n"+
			"%-16s %s\n",
		stateStr,
		term.FormatDuration(time.Since(startTime)),
		"Task Runs:", term.Highlight(result.DeletedTaskRuns),
		"Tasks:", term.Highlight(result.DeletedTasks),
		"Apps:", term.Highlight(result.DeletedApps),
		"Inputs:", term.Highlight(result.DeletedInputs),
		"Outputs:", term.Highlight(result.DeletedOutputs),
		"Uploads:", term.Highlight(result.DeletedUploads),
		"VCSs:", term.Highlight(result.DeletedVCS),
	)

	return err
}
