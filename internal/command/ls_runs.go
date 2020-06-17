package command

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur/format"
	"github.com/simplesurance/baur/format/csv"
	"github.com/simplesurance/baur/format/table"
	"github.com/simplesurance/baur/internal/command/flag"
	"github.com/simplesurance/baur/internal/command/terminal"
	"github.com/simplesurance/baur/log"
	"github.com/simplesurance/baur/storage"
)

const lsRunsLongHelp = `
List  recorded task runs.

Arguments:
	'*' can be passed as <APP-NAME> and <TASK-NAME> argument to match
	all Apps and Tasks.
`

const lsRunsExample = `
baur ls runs -s duration-desc calc               list task runs of the calc
						 application, sorted by
						 run duration
baur ls runs --csv --after=2018.09.27-11:30 '*'  list all task runs in csv format that
						 started after 2018.09.27 11:30`

func init() {
	lsCmd.AddCommand(&newLsRunsCmd().Command)
}

type lsRunsCmd struct {
	cobra.Command

	csv    bool
	after  flag.DateTimeFlagValue
	before flag.DateTimeFlagValue
	sort   *flag.Sort
	quiet  bool

	app  string
	task string
}

func newLsRunsCmd() *lsRunsCmd {
	cmd := lsRunsCmd{
		Command: cobra.Command{
			Use:     "runs <APP-NAME>[.<TASK-NAME>]",
			Short:   "list record task runs",
			Long:    strings.TrimSpace(lsRunsLongHelp),
			Example: strings.TrimSpace(lsRunsExample),
			Args:    cobra.ExactArgs(1),
		},

		sort: flag.NewSort(map[string]storage.Field{
			"time":     storage.FieldStartTime,
			"duration": storage.FieldDuration,
		}),
	}

	cmd.Run = cmd.run

	cmd.Flags().BoolVar(&cmd.csv, "csv", false,
		"List runs in RFC4180 CSV format")

	cmd.Flags().BoolVarP(&cmd.quiet, "quiet", "q", false,
		"Only print task run IDs")

	cmd.Flags().VarP(cmd.sort, "sort", "s",
		cmd.sort.Usage(terminal.Highlight))

	cmd.Flags().VarP(&cmd.after, "after", "a",
		fmt.Sprintf("Only show runs that were started after this datetime.\nFormat: %s", terminal.Highlight(flag.DateTimeFormatDescr)))

	cmd.Flags().VarP(&cmd.before, "before", "b",
		fmt.Sprintf("Only show runs that were started before this datetime.\nFormat: %s", terminal.Highlight(flag.DateTimeFormatDescr)))

	return &cmd
}

func parseSpec(s string) (app, task string) {
	spl := strings.Split(s, ".")

	switch l := len(spl); l {
	case 1:
		return spl[0], ""
	case 2:
		return spl[0], spl[1]

	default:
		stderr.Printf("invalid argument: %q\n", s)
		os.Exit(1)
	}

	// is never executed because of the default case
	return "", ""
}

func (c *lsRunsCmd) run(cmd *cobra.Command, args []string) {
	var sorters []*storage.Sorter

	defaultSorter := storage.Sorter{
		Field: storage.FieldStartTime,
		Order: storage.OrderDesc,
	}

	c.app, c.task = parseSpec(args[0])

	repo := MustFindRepository()
	psql := mustNewCompatibleStorage(repo)

	var formatter format.Formatter
	if c.csv {
		formatter = csv.New(nil, stdout)
	} else {
		formatter = table.New(nil, stdout)
	}

	if !c.csv && !c.quiet {
		printHeader(formatter)
	}

	filters := c.getFilters()
	if c.sort.Value != (storage.Sorter{}) {
		sorters = append(sorters, &c.sort.Value)
	}

	sorters = append(sorters, &defaultSorter)

	err := psql.TaskRuns(
		ctx,
		filters,
		sorters,
		func(taskRun *storage.TaskRunWithID) error {
			return c.printTaskRun(formatter, taskRun)
		},
	)

	if err != nil {
		if err == storage.ErrNotExist {
			log.Fatalf("no matching task runs exist")
		}

		log.Fatalln(err)
	}

	exitOnErr(formatter.Flush())
}

func printHeader(formatter format.Formatter) {
	exitOnErr(formatter.WriteRow([]interface{}{
		"Id",
		"App",
		"Task",
		"Result",
		"Start Time",
		"Duration (s)",
		"Input Digest",
	}))
}

func (c *lsRunsCmd) printTaskRun(formatter format.Formatter, taskRun *storage.TaskRunWithID) error {
	if c.quiet {
		return formatter.WriteRow([]interface{}{taskRun.ID})
	}

	return formatter.WriteRow([]interface{}{
		strconv.Itoa(taskRun.ID),
		taskRun.ApplicationName,
		taskRun.TaskName,
		taskRun.Result,
		taskRun.StartTimestamp.Format(flag.DateTimeFormatTz),
		terminal.DurationToStrSeconds(
			taskRun.StopTimestamp.Sub(taskRun.StartTimestamp),
		),
		taskRun.TotalInputDigest,
	})
}

func (c *lsRunsCmd) getFilters() []*storage.Filter {
	var filters []*storage.Filter

	if c.app != "" && c.app != "*" {
		filters = append(filters, &storage.Filter{
			Field:    storage.FieldApplicationName,
			Operator: storage.OpEQ,
			Value:    c.app,
		})
	}

	if c.task != "" && c.task != "*" {
		filters = append(filters, &storage.Filter{
			Field:    storage.FieldTaskName,
			Operator: storage.OpEQ,
			Value:    c.task,
		})
	}

	if c.before != (flag.DateTimeFlagValue{}) {
		filters = append(filters, &storage.Filter{
			Field:    storage.FieldStartTime,
			Operator: storage.OpLT,
			Value:    c.before.Time,
		})
	}

	if c.after != (flag.DateTimeFlagValue{}) {
		filters = append(filters, &storage.Filter{
			Field:    storage.FieldStartTime,
			Operator: storage.OpGT,
			Value:    c.after.Time,
		})
	}

	return filters
}
