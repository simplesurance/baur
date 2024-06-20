package command

import (
	"fmt"
	"strings"
	"time"

	"github.com/simplesurance/baur/v4/internal/command/flag"
	"github.com/simplesurance/baur/v4/internal/command/term"

	"github.com/spf13/cobra"
)

type cleanupDbCmd struct {
	cobra.Command
	taskRunsBefore flag.DateTimeFlagValue
	force          bool
	pretend        bool
}

const timeFormat = "02 Jan 06 15:04:05 MST"
const cleanupDbGracetime = time.Second * 5

var cleanupDbLongHelp = fmt.Sprintf(`
Delete old data from the baur database.
The command deletes information about tasks runs that started to run before
a given date from the database. It also removes records that became
dangling because all task runs referencing them were deleted.
Task runs that are referenced by a release are not deleted.

The command can be run without access to the baur repository by specifying the
PostgreSQL URI via the environment variable %s.
`,
	term.Highlight(envVarPSQLURL),
)

const cleanupDbCmdExample = `
baur cleanup db --pretend --task-runs-before=2023.06.01-13:30
`

func init() {
	cleanupCmd.AddCommand(&newCleanupDbCmd().Command)
}

func newCleanupDbCmd() *cleanupDbCmd {
	cmd := cleanupDbCmd{
		Command: cobra.Command{
			Args:    cobra.NoArgs,
			Use:     "db --task-runs-before=DATETIME",
			Long:    strings.TrimSpace(cleanupDbLongHelp),
			Example: strings.TrimSpace(cleanupDbCmdExample),
		},
	}

	cmd.Flags().Var(&cmd.taskRunsBefore, "task-runs-before",
		fmt.Sprintf(
			"delete tasks that ran before DATETIME\nFormat: %s",
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

	if err := cmd.MarkFlagRequired("task-runs-before"); err != nil {
		panic(err)
	}

	cmd.Run = cmd.run

	return &cmd
}

func (c *cleanupDbCmd) run(cmd *cobra.Command, _ []string) {
	var op string
	if c.pretend {
		op = term.Highlight("pretending to delete")
	} else {
		op = term.Highlight("deleting")
	}
	stdout.Printf(
		"%s tasks runs older then %s and dangling records,\n"+
			"tasks runs referenced by releases are kept\n",
		op,
		term.Highlight(c.taskRunsBefore.Format(timeFormat)),
	)

	if !c.force {
		stdout.Printf("starting in %s seconds, press %s to abort\n",
			term.Highlight(cleanupDbGracetime), term.Highlight("CTRL+C"))
		time.Sleep(cleanupDbGracetime)
		stdout.Println("starting deleting...")
	}

	psqlURL, err := postgresqlURL()
	exitOnErr(err)

	storageClt := mustNewCompatibleStorage(psqlURL)
	startTime := time.Now()
	result, err := storageClt.TaskRunsDelete(cmd.Context(), c.taskRunsBefore.Time, c.pretend)
	exitOnErr(err)

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
		term.GreenHighlight("successful"),
		term.FormatDuration(time.Since(startTime)),
		"Task Runs:", term.Highlight(result.DeletedTaskRuns),
		"Tasks:", term.Highlight(result.DeletedTasks),
		"Apps:", term.Highlight(result.DeletedApps),
		"Inputs:", term.Highlight(result.DeletedInputs),
		"Outputs:", term.Highlight(result.DeletedOutputs),
		"Uploads:", term.Highlight(result.DeletedUploads),
		"VCSs:", term.Highlight(result.DeletedVCS),
	)
}
