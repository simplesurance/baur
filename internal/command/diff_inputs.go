package command

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur/v1"
	"github.com/simplesurance/baur/v1/internal/digest"
	"github.com/simplesurance/baur/v1/internal/format"
	"github.com/simplesurance/baur/v1/internal/format/csv"
	"github.com/simplesurance/baur/v1/internal/format/table"
	"github.com/simplesurance/baur/v1/storage"
)

type storageInput struct {
	input *storage.Input
}

func (i *storageInput) Digest() (*digest.Digest, error) {
	return digest.FromString(i.input.Digest)
}

func (i *storageInput) String() string {
	return i.input.URI
}

type diffInputArgDetails struct {
	arg      string
	appName  string
	taskName string
	runID    string
	task     *baur.Task
}

func init() {
	diffCmd.AddCommand(&newDiffInputsCmd().Command)
}

type diffInputsCmd struct {
	cobra.Command

	csv      bool
	quiet    bool
	inputStr string
}

const diffInputslongHelp = `
List the difference of inputs between tasks or task-runs.

States:
	D - digests do not match,
	+ - the input is missing in the first task(-run)
	- - the input is missing in the second task(-run)

Exit Codes:
	0 - Inputs are the same
	1 - Internal error
	2 - Inputs differ
`

func newDiffInputsCmd() *diffInputsCmd {
	cmd := diffInputsCmd{
		Command: cobra.Command{
			Use:   "inputs <APP-NAME>.<TASK-NAME>|<RUN-ID> <APP-NAME>.<TASK-NAME>|<RUN-ID>",
			Short: "list inputs that differ between two task-runs",
			Long:  strings.TrimSpace(diffInputslongHelp),
			Args:  diffArgs(),
		},
	}

	cmd.Run = cmd.run

	cmd.Flags().BoolVar(&cmd.csv, "csv", false,
		"show output in RFC4180 CSV format")

	cmd.Flags().BoolVarP(&cmd.quiet, "quiet", "q", false,
		"do not list the inputs that differ")

	cmd.Flags().StringVar(&cmd.inputStr, "input-str", "",
		"include a string as input")

	return &cmd
}

// diffArgs returns an error in the following scenarios:
// - there is less than or greater than 2 args specified
// - the <APP-NAME> or <TASK-NAME> is a wildcard character (*)
// - either arg is not in the format <APP-NAME>.<TASK-NAME> or a numeric value
func diffArgs() cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) != 2 {
			return fmt.Errorf("accepts 2 args, received %d", len(args))
		}

		validArgRE := regexp.MustCompile(`^\w+\.[\w\^]+$|^[0-9]+\d*$`)
		for _, arg := range args {
			if !validArgRE.MatchString(arg) {
				return fmt.Errorf("invalid argument: %q", arg)
			}
		}
		return nil
	}
}

func (c *diffInputsCmd) run(cmd *cobra.Command, args []string) {
	if args[0] == args[1] {
		exitOnErr(fmt.Errorf("%s and %s refer to the same task-run", args[0], args[1]))
	}

	repo := mustFindRepository()
	argDetails := getDiffInputArgDetails(repo, args)

	inputs1, run1 := c.getTaskRunInputs(repo, argDetails[0])
	inputs2, run2 := c.getTaskRunInputs(repo, argDetails[1])

	if run1 != nil && run2 != nil {
		if run1.ID == run2.ID {
			exitOnErr(fmt.Errorf("%s and %s refer to the same task-run", args[0], args[1]))
		}
	}

	diffs, err := baur.DiffInputs(inputs1, inputs2)
	exitOnErr(err)

	c.printOutput(diffs)

	if len(diffs) > 0 {
		exitFunc(2)
	}

	exitFunc(0)
}

func getDiffInputArgDetails(repo *baur.Repository, args []string) []*diffInputArgDetails {
	results := make([]*diffInputArgDetails, 0, len(args))

	for _, arg := range args {
		app, task, runID := parseDiffSpec(arg)
		results = append(results, &diffInputArgDetails{arg: arg, appName: app, taskName: task, runID: runID})
	}

	var mustHaveTasks []string
	for _, argDetails := range results {
		if argDetails.runID == "" {
			mustHaveTasks = append(mustHaveTasks, argDetails.arg)
		}
	}

	if len(mustHaveTasks) > 0 {
		tasks := mustArgToTasks(repo, mustHaveTasks)

		for _, task := range tasks {
			for _, argDetails := range results {
				if argDetails.runID == "" && task.AppName == argDetails.appName && task.Name == argDetails.taskName {
					argDetails.task = task
				}
			}
		}

		for _, argDetails := range results {
			if argDetails.runID == "" && argDetails.task == nil {
				exitOnErr(fmt.Errorf("task not found for %s", argDetails.arg))
			}
		}
	}

	return results
}

// parseDiffSpec splits an argument into the app, task and runID components.
// It relies on the fact that the arguments have already been validated
// in the cobra.Command
func parseDiffSpec(s string) (app, task, runID string) {
	if strings.Contains(s, ".") {
		spl := strings.Split(s, ".")
		app = spl[0]
		task = spl[1]

		if strings.Contains(task, "^") {
			caretIndex := strings.Index(task, "^")
			runID = subStr(task, caretIndex, len(task))
			task = subStr(task, 0, caretIndex)
		}

		return app, task, runID
	}

	return "", "", s
}

func (c *diffInputsCmd) getTaskRunInputs(repo *baur.Repository, argDetails *diffInputArgDetails) (*baur.Inputs, *storage.TaskRunWithID) {
	if argDetails.task != nil {
		inputResolver := baur.NewInputResolver()

		inputFiles, err := inputResolver.Resolve(ctx, repo.Path, argDetails.task)
		exitOnErr(err)

		return baur.NewInputs(baur.InputAddStrIfNotEmpty(inputFiles, c.inputStr)), nil
	}

	taskRun := getTaskRun(repo, argDetails)

	psql := mustNewCompatibleStorage(repo)
	storageInputs, err := psql.Inputs(ctx, taskRun.ID)
	exitOnErr(err)

	// Convert the inputs from the DB into baur.Input interface implementation
	var baurInputs []baur.Input
	for _, input := range storageInputs {
		baurInputs = append(baurInputs, &storageInput{input})
	}

	return baur.NewInputs(baur.InputAddStrIfNotEmpty(baurInputs, c.inputStr)), taskRun
}

func getTaskRun(repo *baur.Repository, argDetails *diffInputArgDetails) *storage.TaskRunWithID {
	psql := mustNewCompatibleStorage(repo)

	if strings.Contains(argDetails.runID, "^") {
		return getPreviousTaskRun(repo, psql, argDetails)
	}

	id, err := strconv.Atoi(argDetails.runID)
	exitOnErr(err)

	return getTaskRunByID(repo, psql, id)
}

func getPreviousTaskRun(repo *baur.Repository, psql storage.Storer, argDetails *diffInputArgDetails) *storage.TaskRunWithID {
	var filters []*storage.Filter

	filters = append(filters, &storage.Filter{
		Field:    storage.FieldApplicationName,
		Operator: storage.OpEQ,
		Value:    argDetails.appName,
	})

	filters = append(filters, &storage.Filter{
		Field:    storage.FieldTaskName,
		Operator: storage.OpEQ,
		Value:    argDetails.taskName,
	})

	var sorters = []*storage.Sorter{
		{
			Field: storage.FieldID,
			Order: storage.OrderDesc,
		},
	}

	var taskRun *storage.TaskRunWithID
	found := errors.New("found_task_run")
	runPosition := len(argDetails.runID)
	retrieved := 0
	err := psql.TaskRuns(
		ctx,
		filters,
		sorters,
		func(record *storage.TaskRunWithID) error {
			retrieved++
			if retrieved == runPosition {
				taskRun = record
				return found
			}
			return nil
		},
	)

	if err != nil && errors.Unwrap(err) != found {
		exitOnErr(err)
	}

	if runPosition > retrieved {
		exitOnErr(fmt.Errorf("%s does not exist, only %d task-run(s) exist(s)", argDetails.arg, retrieved))
	}

	return taskRun
}

func getTaskRunByID(repo *baur.Repository, psql storage.Storer, id int) *storage.TaskRunWithID {
	filters := []*storage.Filter{
		{
			Field:    storage.FieldID,
			Operator: storage.OpEQ,
			Value:    id,
		},
	}

	sorters := []*storage.Sorter{
		{
			Field: storage.FieldID,
			Order: storage.OrderDesc,
		},
	}

	var taskRun *storage.TaskRunWithID
	err := psql.TaskRuns(
		ctx,
		filters,
		sorters,
		func(run *storage.TaskRunWithID) error {
			taskRun = run
			return nil
		},
	)

	if err != nil && err == storage.ErrNotExist {
		err = fmt.Errorf("task-run %d does not exist", id)
	}
	exitOnErr(err)

	return taskRun
}

func (c *diffInputsCmd) printOutput(diffs []*baur.InputDiff) {
	if !c.quiet {
		var formatter format.Formatter

		if c.csv {
			formatter = csv.New(nil, stdout)
		} else {
			headers := []string{"State", "Path", "Digest1", "Digest2"}
			formatter = table.New(headers, stdout)
		}

		for _, diff := range diffs {
			mustWriteRow(formatter, diff.State, diff.Path, diff.Digest1, diff.Digest2)
		}

		err := formatter.Flush()
		exitOnErr(err)
	}

	if c.csv {
		return
	}

	if !c.quiet {
		stdout.Println()
	}

	if len(diffs) > 0 {
		stdout.Printf("the inputs differ\n")
	} else {
		stdout.Printf("the inputs are the same\n")
	}
}
