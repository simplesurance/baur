package command

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur/v1"
	"github.com/simplesurance/baur/v1/internal/command/flag"
	"github.com/simplesurance/baur/v1/internal/command/term"
	"github.com/simplesurance/baur/v1/internal/format"
	"github.com/simplesurance/baur/v1/internal/format/csv"
	"github.com/simplesurance/baur/v1/internal/format/table"
	"github.com/simplesurance/baur/v1/storage"
)

const lsRunsLongHelp = `
List recorded task runs.

Arguments:
	'*' can be passed as <APP-NAME> or <TASK-NAME> argument to match
	all Apps or Tasks.
`

const lsRunsExample = `
baur ls runs -s duration-desc calc               list task runs of the calc
						 application, sorted by
						 run duration
baur ls runs --csv --after=2018.09.27-11:30 '*'  list all task runs in csv format that
						 were started after 2018.09.27 11:30
baur ls runs --limit=1 calc                      list a single task run of the calc
						 application`

func init() {
	lsCmd.AddCommand(&newLsRunsCmd().Command)
}

type lsRunsCmd struct {
	cobra.Command

	csv      bool
	after    flag.DateTimeFlagValue
	before   flag.DateTimeFlagValue
	inputStr string
	sort     *flag.Sort
	limit    int
	quiet    bool

	app  string
	task string
}

func newLsRunsCmd() *lsRunsCmd {
	cmd := lsRunsCmd{
		Command: cobra.Command{
			Use:     "runs <APP-NAME>[.<TASK-NAME>]",
			Short:   "list recorded task runs",
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
		cmd.sort.Usage(term.Highlight))

	cmd.Flags().IntVarP(&cmd.limit, "limit", "l", storage.NoLimit,
		"Limit the number of runs shown, 0 will show all runs")

	cmd.Flags().VarP(&cmd.after, "after", "a",
		fmt.Sprintf("Only show runs that were started after this datetime.\nFormat: %s", term.Highlight(flag.DateTimeFormatDescr)))

	cmd.Flags().VarP(&cmd.before, "before", "b",
		fmt.Sprintf("Only show runs that were started before this datetime.\nFormat: %s", term.Highlight(flag.DateTimeFormatDescr)))

	cmd.Flags().StringVar(&cmd.inputStr, "has-input-str", "",
		"Only show runs that include this value as an input-str value")

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
		exitFunc(1)
	}

	// is never executed because of the default case
	panic("default case not run")
}

func (c *lsRunsCmd) run(cmd *cobra.Command, args []string) {
	var sorters []*storage.Sorter

	defaultSorter := storage.Sorter{
		Field: storage.FieldStartTime,
		Order: storage.OrderDesc,
	}

	c.app, c.task = parseSpec(args[0])

	repo := mustFindRepository()
	psql := mustNewCompatibleStorage(repo)
	defer psql.Close()

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

	var err error
	if c.inputStr != "" {
		err = psql.TaskRunsWithInputURI(
			ctx,
			filters,
			sorters,
			c.limit,
			baur.NewInputString(c.inputStr).String(),
			func(taskRun *storage.TaskRunWithID) error {
				c.printTaskRun(formatter, taskRun)
				return nil
			},
		)
	} else {
		err = psql.TaskRuns(
			ctx,
			filters,
			sorters,
			c.limit,
			func(taskRun *storage.TaskRunWithID) error {
				c.printTaskRun(formatter, taskRun)
				return nil
			},
		)
	}

	if err != nil {
		if err == storage.ErrNotExist {
			stderr.Printf("no matching task runs exist")
			exitFunc(1)
		}
		stderr.Println(err)
		exitFunc(1)
	}

	exitOnErr(formatter.Flush())
}

func printHeader(formatter format.Formatter) {
	mustWriteRow(
		formatter,
		"Id",
		"App",
		"Task",
		"Result",
		"Start Time",
		"Duration",
		"Input Digest",
	)
}

func (c *lsRunsCmd) printTaskRun(formatter format.Formatter, taskRun *storage.TaskRunWithID) {
	if c.quiet {
		mustWriteRow(formatter, taskRun.ID)
	}

	mustWriteRow(formatter,
		strconv.Itoa(taskRun.ID),
		taskRun.ApplicationName,
		taskRun.TaskName,
		taskRun.Result,
		taskRun.StartTimestamp.Format(flag.DateTimeFormatTz),
		term.FormatDuration(
			taskRun.StopTimestamp.Sub(taskRun.StartTimestamp),
			term.FormatBaseWithoutUnitName(c.csv),
		),
		taskRun.TotalInputDigest,
	)
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
