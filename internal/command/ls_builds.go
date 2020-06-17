package command

import (
	"fmt"
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

const lsBuildsExample = `
baur ls builds -s duration-desc calc               list builds of the calc
						   application, sorted by
						   build duration
baur ls builds --csv --after=2018.09.27-11:30 all  list builds in csv format that
						   happened after 2018.09.27 11:30`

var lsBuildsCmd = &cobra.Command{
	Use:     "builds <APP-NAME>|all",
	Short:   "list builds for an application",
	Example: strings.TrimSpace(lsBuildsExample),
	Args:    cobra.ExactArgs(1),
	Run:     runBuildLs,
}

type lsBuildsConf struct {
	app    string
	csv    bool
	after  flag.DateTimeFlagValue
	before flag.DateTimeFlagValue
	sort   *flag.Sort
	quiet  bool
}

var lsBuildsConfig lsBuildsConf

func init() {
	lsBuildsConfig.sort = flag.NewSort(map[string]storage.Field{
		"time":     storage.FieldStartTime,
		"duration": storage.FieldDuration,
	})

	lsBuildsCmd.Flags().BoolVar(&lsBuildsConfig.csv, "csv", false,
		"List builds in RFC4180 CSV format")

	lsBuildsCmd.Flags().BoolVarP(&lsBuildsConfig.quiet, "quiet", "q", false,
		"Only print build IDs")

	lsBuildsCmd.Flags().VarP(lsBuildsConfig.sort, "sort", "s",
		lsBuildsConfig.sort.Usage(terminal.Highlight))

	lsBuildsCmd.Flags().VarP(&lsBuildsConfig.after, "after", "a",
		fmt.Sprintf("Only show builds that were build after this datetime.\nFormat: %s", terminal.Highlight(flag.DateTimeFormatDescr)))

	lsBuildsCmd.Flags().VarP(&lsBuildsConfig.before, "before", "b",
		fmt.Sprintf("Only show builds that were build before this datetime.\nFormat: %s", terminal.Highlight(flag.DateTimeFormatDescr)))

	lsCmd.AddCommand(lsBuildsCmd)
}

func runBuildLs(cmd *cobra.Command, args []string) {
	var sorters []*storage.Sorter

	defaultSorter := storage.Sorter{
		Field: storage.FieldStartTime,
		Order: storage.OrderDesc,
	}

	lsBuildsConfig.app = args[0]

	repo := MustFindRepository()
	psql := mustNewCompatibleStorage(repo)

	var formatter format.Formatter
	if lsBuildsConfig.csv {
		formatter = csv.New(nil, stdout)
	} else {
		formatter = table.New(nil, stdout)
	}

	if !lsBuildsConfig.csv && !lsBuildsConfig.quiet {
		printHeader(formatter)
	}

	filters := lsBuildsConfig.getFilters()
	if lsBuildsConfig.sort.Value != (storage.Sorter{}) {
		sorters = append(sorters, &lsBuildsConfig.sort.Value)
	}

	sorters = append(sorters, &defaultSorter)

	err := psql.TaskRuns(
		ctx,
		filters,
		sorters,
		func(taskRun *storage.TaskRunWithID) error {
			return printTaskRun(formatter, taskRun)
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
		"Start Time",
		"Duration (s)",
		"Input Digest",
	}))
}

func printTaskRun(formatter format.Formatter, taskRun *storage.TaskRunWithID) error {
	if lsBuildsConfig.quiet {
		return formatter.WriteRow([]interface{}{taskRun.ID})
	}

	return formatter.WriteRow([]interface{}{
		strconv.Itoa(taskRun.ID),
		taskRun.ApplicationName,
		taskRun.StartTimestamp.Format(flag.DateTimeFormatTz),
		terminal.DurationToStrSeconds(
			taskRun.StopTimestamp.Sub(taskRun.StartTimestamp),
		),
		taskRun.TotalInputDigest,
	})
}

func (conf lsBuildsConf) getFilters() []*storage.Filter {
	var filters []*storage.Filter

	if conf.app != "all" {
		filters = append(filters, &storage.Filter{
			Field:    storage.FieldApplicationName,
			Operator: storage.OpEQ,
			Value:    conf.app,
		})
	}

	if conf.before != (flag.DateTimeFlagValue{}) {
		filters = append(filters, &storage.Filter{
			Field:    storage.FieldStartTime,
			Operator: storage.OpLT,
			Value:    conf.before.Time,
		})
	}

	if conf.after != (flag.DateTimeFlagValue{}) {
		filters = append(filters, &storage.Filter{
			Field:    storage.FieldStartTime,
			Operator: storage.OpGT,
			Value:    conf.after.Time,
		})
	}

	return filters
}
