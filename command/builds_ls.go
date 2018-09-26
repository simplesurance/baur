package command

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/simplesurance/baur/command/flag"
	"github.com/simplesurance/baur/log"
	"github.com/simplesurance/baur/storage"
	"github.com/simplesurance/baur/storage/postgres"
	viewList "github.com/simplesurance/baur/view/list"
	"github.com/simplesurance/baur/view/list/dataprovider"
)

const buildsLsExample = `
baur builds ls -s duration-desc calc               list builds of the calc
						   application, sorted by build duration
baur builds ls --csv --after=2018-09-27-11:30 all  list builds in csv format that
					           happened after 2018.09.27 11:30`

const buildsLsLongHelp = `
baur builds ls allows to list builds of applications
`

var buildsLsSortHelp = fmt.Sprintf(`
Sort the list by a specific field.
Format: %s
where %s is one of: %s, %s
and %s one of: %s, %s`,
	highlight("<FIELD>-<ORDER>"),
	highlight("<FIELD>"), highlight("time"),
	highlight("duration"),
	highlight("<ORDER>"), highlight("asc"), highlight("desc"))

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
	sort   flag.SortFlagValue
	quiet  bool
}

var buildsLsConfig buildsLsConf

// highlight is a function that highlights parts of strings in the cli output
var highlight = color.New(color.FgGreen).SprintFunc()

func init() {

	buildsLsCmd.Flags().BoolVar(&buildsLsConfig.csv, "csv", false,
		"Lists applications in the RFC4180 CSV format")

	buildsLsCmd.Flags().BoolVarP(&buildsLsConfig.quiet, "quiet", "q", false,
		"Only print builds ids")

	buildsLsCmd.Flags().VarP(&buildsLsConfig.sort, "sort", "s",
		strings.TrimSpace(buildsLsSortHelp))

	buildsLsCmd.Flags().VarP(&buildsLsConfig.after, "after", "f",
		fmt.Sprintf("Only show builds that were build after this datetime.\nFormat: %s", highlight(flag.DateTimeFormatDescr)))

	buildsLsCmd.Flags().VarP(&buildsLsConfig.before, "before", "b",
		fmt.Sprintf("Only show builds that were build before this datetime.\nFormat: %s", highlight(flag.DateTimeFormatDescr)))

	buildsCmd.AddCommand(buildsLsCmd)
}

func runBuildLs(cmd *cobra.Command, args []string) {
	buildsLsConfig.app = args[0]

	repo := MustFindRepository()

	listProvider := dataprovider.NewBuildListProvider(MustGetPostgresClt(repo))

	filters := buildsLsConfig.getFilters()

	sorters, err := buildsLsConfig.getSorters()
	if err != nil {
		log.Fatalln(errors.Wrap(err, "invalid sorter string"))
	}

	err = listProvider.FetchData(filters, sorters)
	if err != nil {
		log.Fatalln(errors.Wrap(err, "fetching data failed"))
	}

	list := viewList.NewList(
		[]*viewList.Column{
			{Name: "Id"},
			{Name: "App"},
			{Name: "Start Time"},
			{Name: "Duration (s)"},
			{Name: "Input Digest"},
		},
		listProvider,
	)

	var flattener viewList.FlattenerFunc
	if buildsLsConfig.csv {
		flattener = viewList.CsvListFlattener
	} else {
		flattener = viewList.DefaultListFlattener
	}

	str, err := list.Flatten(flattener, highlight, buildsLsConfig.quiet)
	if err != nil {
		log.Fatalln("formatting data failed: ", err.Error())
	}

	fmt.Println(str)
}

func (conf buildsLsConf) getFilters() (filters []storage.CanFilter) {
	if conf.app != "all" {
		filter := postgres.NewFilter(storage.FieldApplicationName, storage.OperatorEq, conf.app)
		filters = append(filters, filter)
	}

	if conf.before != (flag.DateTimeFlagValue{}) {
		filter := postgres.NewFilter(storage.FieldBuildStartDatetime, storage.OperatorLte, conf.before.Time)
		filters = append(filters, filter)
	}

	if conf.after != (flag.DateTimeFlagValue{}) {
		filter := postgres.NewFilter(storage.FieldBuildStartDatetime, storage.OperatorGte, conf.after.Time)
		filters = append(filters, filter)
	}

	return
}

func (conf buildsLsConf) getSorters() (sorters []storage.CanSort, err error) {
	// TODO: does this function really have to return an error value, it
	// never fails
	if conf.sort != (flag.SortFlagValue{}) {
		sorters = append(sorters, conf.sort.Sorter)
	}

	return
}
