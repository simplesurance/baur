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

const buildsLsExample = `
baur builds ls -s duration-desc calc               list builds of the calc
						   application, sorted by build duration
baur builds ls --csv --after=2018.09.27-11:30 all  list builds in csv format that
					           happened after 2018.09.27 11:30`

const buildsLsLongHelp = `
baur builds ls allows to list builds of applications
`

var buildsLsCmd = &cobra.Command{
	Use:     "ls <APP-NAME>|all",
	Short:   "list builds",
	Long:    strings.TrimSpace(buildsLsLongHelp),
	Example: strings.TrimSpace(buildsLsExample),
	Args:    cobra.ExactArgs(1),
	Run:     runBuildLs,
}

type buildsLsConf struct {
	app    string
	csv    bool
	after  flag.DateTimeFlagValue
	before flag.DateTimeFlagValue
	sort   *flag.Sort
	quiet  bool
}

var buildsLsConfig buildsLsConf

func init() {
	buildsLsConfig.sort = flag.NewSort(map[string]storage.Field{
		"time":     storage.FieldBuildStartTime,
		"duration": storage.FieldBuildDuration,
	})

	buildsLsCmd.Flags().BoolVar(&buildsLsConfig.csv, "csv", false,
		"Lists applications in the RFC4180 CSV format")

	buildsLsCmd.Flags().BoolVarP(&buildsLsConfig.quiet, "quiet", "q", false,
		"Only print builds ids")

	buildsLsCmd.Flags().VarP(buildsLsConfig.sort, "sort", "s",
		buildsLsConfig.sort.Usage(highlight))

	buildsLsCmd.Flags().VarP(&buildsLsConfig.after, "after", "a",
		fmt.Sprintf("Only show builds that were build after this datetime.\nFormat: %s", highlight(flag.DateTimeFormatDescr)))

	buildsLsCmd.Flags().VarP(&buildsLsConfig.before, "before", "b",
		fmt.Sprintf("Only show builds that were build before this datetime.\nFormat: %s", highlight(flag.DateTimeFormatDescr)))

	buildsCmd.AddCommand(buildsLsCmd)
}

func runBuildLs(cmd *cobra.Command, args []string) {
	var sorters []*storage.Sorter

	defaultSorter := storage.Sorter{
		Field: storage.FieldBuildStartTime,
		Order: storage.OrderDesc,
	}

	buildsLsConfig.app = args[0]

	repo := MustFindRepository()

	filters := buildsLsConfig.getFilters()
	if buildsLsConfig.sort.Value != (storage.Sorter{}) {
		sorters = append(sorters, &buildsLsConfig.sort.Value)
	}

	sorters = append(sorters, &defaultSorter)

	printBuilds(repo, filters, sorters)
}

func printBuilds(repo *baur.Repository, filters []*storage.Filter, sorters []*storage.Sorter) {
	var headers []string
	var formatter format.Formatter
	psql := MustGetPostgresClt(repo)
	writeHeaders := !buildsLsConfig.quiet

	if writeHeaders {
		headers = []string{
			"Id",
			"App",
			"Start Time",
			"Duration (s)",
			"Input Digest",
		}

	}

	if buildsLsConfig.csv {
		formatter = csv.New(headers, os.Stdout, writeHeaders)
	} else {
		formatter = table.New(headers, os.Stdout, writeHeaders)
	}

	builds, err := psql.GetBuilds(filters, sorters)
	if err != nil {
		log.Fatalln(err)
	}

	for _, build := range builds {
		var row format.Row

		if buildsLsConfig.quiet {
			row.Data = []interface{}{build.ID}
		} else {
			row.Data = []interface{}{
				strconv.Itoa(build.ID),
				build.Application.Name,
				build.StartTimeStamp.Format(flag.DateTimeFormatTz),
				fmt.Sprint(build.Duration.Seconds()),
				build.TotalInputDigest,
			}
		}

		if err := formatter.WriteRow(&row); err != nil {
			log.Fatalln(err)
		}

	}

	if err := formatter.Flush(); err != nil {
		log.Fatalln(err)
	}
}

func (conf buildsLsConf) getFilters() (filters []*storage.Filter) {
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
