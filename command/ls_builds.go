package command

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur"
	"github.com/simplesurance/baur/command/flag"
	"github.com/simplesurance/baur/format"
	"github.com/simplesurance/baur/format/csv"
	"github.com/simplesurance/baur/format/table"
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
		"time":     storage.FieldBuildStartTime,
		"duration": storage.FieldBuildDuration,
	})

	lsBuildsCmd.Flags().BoolVar(&lsBuildsConfig.csv, "csv", false,
		"List builds in RFC4180 CSV format")

	lsBuildsCmd.Flags().BoolVarP(&lsBuildsConfig.quiet, "quiet", "q", false,
		"Only print build IDs")

	lsBuildsCmd.Flags().VarP(lsBuildsConfig.sort, "sort", "s",
		lsBuildsConfig.sort.Usage(highlight))

	lsBuildsCmd.Flags().VarP(&lsBuildsConfig.after, "after", "a",
		fmt.Sprintf("Only show builds that were build after this datetime.\nFormat: %s", highlight(flag.DateTimeFormatDescr)))

	lsBuildsCmd.Flags().VarP(&lsBuildsConfig.before, "before", "b",
		fmt.Sprintf("Only show builds that were build before this datetime.\nFormat: %s", highlight(flag.DateTimeFormatDescr)))

	lsCmd.AddCommand(lsBuildsCmd)
}

func runBuildLs(cmd *cobra.Command, args []string) {
	var sorters []*storage.Sorter

	defaultSorter := storage.Sorter{
		Field: storage.FieldBuildStartTime,
		Order: storage.OrderDesc,
	}

	lsBuildsConfig.app = args[0]

	repo := MustFindRepository()

	filters := lsBuildsConfig.getFilters()
	if lsBuildsConfig.sort.Value != (storage.Sorter{}) {
		sorters = append(sorters, &lsBuildsConfig.sort.Value)
	}

	sorters = append(sorters, &defaultSorter)

	printBuilds(repo, filters, sorters)
}

func printBuilds(repo *baur.Repository, filters []*storage.Filter, sorters []*storage.Sorter) {
	var headers []string
	var formatter format.Formatter
	psql := MustGetPostgresClt(repo)
	writeHeaders := !lsBuildsConfig.quiet && !lsBuildsConfig.csv

	if writeHeaders {
		headers = []string{
			"Id",
			"App",
			"Start Time",
			"Duration (s)",
			"Input Digest",
		}

	}

	if lsBuildsConfig.csv {
		formatter = csv.New(headers, os.Stdout)
	} else {
		formatter = table.New(headers, os.Stdout)
	}

	builds, err := psql.GetBuilds(filters, sorters)
	if err != nil {
		log.Fatalln(err)
	}

	for _, build := range builds {
		var row []interface{}

		if lsBuildsConfig.quiet {
			row = []interface{}{build.ID}
		} else {
			row = []interface{}{
				strconv.Itoa(build.ID),
				build.Application.Name,
				build.StartTimeStamp.Format(flag.DateTimeFormatTz),
				fmt.Sprint(build.Duration.Seconds()),
				build.TotalInputDigest,
			}
		}

		if err := formatter.WriteRow(row); err != nil {
			log.Fatalln(err)
		}

	}

	if err := formatter.Flush(); err != nil {
		log.Fatalln(err)
	}
}

func (conf lsBuildsConf) getFilters() (filters []*storage.Filter) {
	if conf.app != "all" {
		filters = append(filters, &storage.Filter{
			Field:    storage.FieldApplicationName,
			Operator: storage.OpEQ,
			Value:    conf.app,
		})
	}

	if conf.before != (flag.DateTimeFlagValue{}) {
		filters = append(filters, &storage.Filter{
			Field:    storage.FieldBuildStartTime,
			Operator: storage.OpLT,
			Value:    conf.before.Time,
		})
	}

	if conf.after != (flag.DateTimeFlagValue{}) {
		filters = append(filters, &storage.Filter{
			Field:    storage.FieldBuildStartTime,
			Operator: storage.OpGT,
			Value:    conf.after.Time,
		})
	}

	return
}
