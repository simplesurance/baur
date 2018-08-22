package command

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/simplesurance/baur/command/flag"
	"github.com/simplesurance/baur/log"
	"github.com/simplesurance/baur/storage"
	"github.com/simplesurance/baur/storage/postgres"
	viewList "github.com/simplesurance/baur/view/list"
	"github.com/simplesurance/baur/view/list/data_provider"
	"github.com/spf13/cobra"
)

var buildLsCmd = &cobra.Command{
	Use:   "ls",
	Short: "list builds",
	Long:  "Lists builds and provides a set and filters and sorters for this list",
	Example: "\nls -n 2018-08-06T00:00:00 -o 2018-08-07T00:00:00 -a acl-service -q\n" +
		"\nls --sort duration-desc acl-service",
	Args:      argsBuildLs,
	Run:       runBuildLs,
	ValidArgs: []string{},
}

type buildLsConf struct {
	csv    bool
	app    string
	oldest flag.DateTimeFlagValue
	newest flag.DateTimeFlagValue
	sort   flag.SortFlagValue
	quiet  bool
}

var buildLsConfig buildLsConf

// highlight is a function that highlights parts of strings in the cli output
var highlight viewList.StringHighlighterFunc

func init() {
	highlight = color.New(color.FgGreen).SprintFunc()

	buildLsCmd.Flags().BoolVar(&buildLsConfig.csv, "csv", false,
		"Lists applications in the RFC4180 CSV format")

	buildLsCmd.Flags().BoolVarP(&buildLsConfig.quiet, "quiet", "q", false,
		"Only shows ids. Useful for quickly exporting or piping lists")

	buildLsCmd.Flags().StringVarP(&buildLsConfig.app, "app", "a", "",
		"Only show the builds corresponding to this app name")

	buildLsCmd.Flags().VarP(&buildLsConfig.sort, "sort", "s",
		fmt.Sprintf("Sorts the list by %s or %s, using the following format: %s. Examples: %s, %s",
			highlight("time"),
			highlight("duration"),
			highlight("<sort_by>-<sort_direction>"),
			highlight("time-asc"),
			highlight("duration-desc"),
		),
	)

	buildLsCmd.Flags().VarP(&buildLsConfig.newest, "newest", "n",
		fmt.Sprintf("only show builds older than this one. Example: %s", highlight(flag.InputTimeFormat)))

	buildLsCmd.Flags().VarP(&buildLsConfig.oldest, "oldest", "o",
		fmt.Sprintf("only show builds newer than this one. Example: %s", highlight(flag.InputTimeFormat)))

	buildCmd.AddCommand(buildLsCmd)
}

func argsBuildLs(cmd *cobra.Command, args []string) error {
	err := cobra.MaximumNArgs(1)(cmd, args)
	if err != nil {
		return err
	}

	if len(args) > 0 {
		// if present, use the first arg as the app name filter
		buildLsConfig.app = args[0]
	}

	return nil
}

func runBuildLs(cmd *cobra.Command, args []string) {
	repo := MustFindRepository()

	listProvider := data_provider.NewBuildListProvider(MustGetPostgresClt(repo))

	filters := buildLsConfig.getFilters()

	sorters, err := buildLsConfig.getSorters()
	if err != nil {
		log.Fatalln(errors.Wrap(err, "invalid sorter string"))
	}

	err = listProvider.FetchData(filters, sorters)
	if err != nil {
		log.Fatalln(errors.Wrap(err, "problem while fetching data"))
	}

	list := viewList.NewList(
		[]*viewList.Column{
			{"Id"},
			{"App"},
			{"Start Time"},
			{"Duration (s)"},
			{"Input Digest"},
		},
		listProvider,
	)

	var flattener viewList.FlattenerFunc
	if buildLsConfig.csv {
		flattener = viewList.CsvListFlattener
	} else {
		flattener = viewList.DefaultListFlattener
	}

	str, err := list.Flatten(flattener, highlight, buildLsConfig.quiet)
	if err != nil {
		log.Fatalf("list display error: %s", err.Error())
	}

	fmt.Println(str)
}

func (conf buildLsConf) getFilters() (filters []storage.CanFilter) {
	if conf.app != "" {
		filter := postgres.NewFilter(storage.FieldApplicationName, storage.OperatorEq, conf.app)
		filters = append(filters, filter)
	}

	if conf.newest != (flag.DateTimeFlagValue{}) {
		filter := postgres.NewFilter(storage.FieldBuildStartDatetime, storage.OperatorLte, conf.newest.Time)
		filters = append(filters, filter)
	}

	if conf.oldest != (flag.DateTimeFlagValue{}) {
		filter := postgres.NewFilter(storage.FieldBuildStartDatetime, storage.OperatorGte, conf.oldest.Time)
		filters = append(filters, filter)
	}

	return
}

func (conf buildLsConf) getSorters() (sorters []storage.CanSort, err error) {
	if conf.sort != (flag.SortFlagValue{}) {
		sorters = append(sorters, conf.sort.Sorter)
	}

	return
}
