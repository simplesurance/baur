package command

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur/v3/internal/format/csv"
	"github.com/simplesurance/baur/v3/internal/format/table"
	"github.com/simplesurance/baur/v3/internal/vcs/git"
	"github.com/simplesurance/baur/v3/pkg/baur"
	"github.com/simplesurance/baur/v3/pkg/storage"
)

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
	inputStr []string

	gitRepo *git.Repository
}

const diffInputslongHelp = `
List the difference of inputs between tasks or task-runs.

An argument can either reference a task or a task-run.
If a task is specified the current inputs of the task in the filesystem are
compared. A task is specified in the format APP_NAME.TASK_NAME.
A past task-run that was recorded in the database can be specified by:
- its run-id,
- or by the git-like syntax APP_NAME.TASK_NAME^, the number of ^ characters
  specify the run-id, ^ refers the last recorded run, '^^' the run before the
  last, and so on

States:
	D - digests do not match,
	+ - the input is missing in the first task(-run)
	- - the input is missing in the second task(-run)

Exit Codes:
	0 - Inputs are the same
	1 - Internal error
	2 - Inputs differ
`

const diffInputsExample = `
baur diff inputs calc.build calc.check  - Compare current inputs of the
					  build and check tasks of the calc app
baur diff inputs calc.build 312		- Compare current inputs of the build
					  task of the calc app with the recorded
					  run with ID 312.
baur diff inputs calc.build calc.build^ - Compare current inputs and the one
					  of the last recorded run of the calc
					  task of the build app.
`

func newDiffInputsCmd() *diffInputsCmd {
	cmd := diffInputsCmd{
		Command: cobra.Command{
			Use:     "inputs <APP_NAME.TASK_NAME[^]|RUN_ID> <APP_NAME.TASK_NAME[^]|RUN_ID>",
			Short:   "list inputs that differ between two task-runs",
			Long:    strings.TrimSpace(diffInputslongHelp),
			Example: strings.TrimSpace(diffInputsExample),
			Args:    diffArgs(),
			ValidArgsFunction: newCompleteTargetFunc(completeTargetFuncOpts{
				withoutWildcards: true,
				withoutAppNames:  true,
				withoutPaths:     true,
			}),
		},
	}

	cmd.Run = cmd.run

	cmd.Flags().BoolVar(&cmd.csv, "csv", false,
		"show output in RFC4180 CSV format")

	cmd.Flags().BoolVarP(&cmd.quiet, "quiet", "q", false,
		"do not list the inputs that differ")

	cmd.Flags().StringArrayVar(&cmd.inputStr, "input-str", nil,
		"include a string as input of the task or run specified by the first argument,\n"+
			"can be specified multiple times")

	return &cmd
}

// diffArgs returns an error in the following scenarios:
// - there is less than or greater than 2 args specified
// - either arg is not in the format APP-NAME.TASK-AME> or a numeric value
func diffArgs() cobra.PositionalArgs {
	return func(_ *cobra.Command, args []string) error {
		if len(args) != 2 {
			return fmt.Errorf("accepts 2 args, received %d", len(args))
		}

		validArgRE := regexp.MustCompile(`^[\w-]+\.[\w-]+\^*$|^[0-9]+\d*$`)
		for _, arg := range args {
			if !validArgRE.MatchString(arg) {
				return fmt.Errorf("invalid argument: %q", arg)
			}
		}
		return nil
	}
}

func (c *diffInputsCmd) run(_ *cobra.Command, args []string) {
	if len(c.inputStr) == 0 && args[0] == args[1] {
		exitOnErr(fmt.Errorf("%s and %s refer to the same task-run", args[0], args[1]))
	}

	repo := mustFindRepository()
	c.gitRepo = mustGetRepoState(repo.Path)
	argDetails := c.getDiffInputArgDetails(repo, args)

	inputs1, run1 := c.getTaskRunInputs(repo, argDetails[0], true)
	inputs2, run2 := c.getTaskRunInputs(repo, argDetails[1], false)

	if len(c.inputStr) == 0 && run1 != nil && run2 != nil && run1.ID == run2.ID {
		exitOnErr(fmt.Errorf("%s and %s refer to the same task-run", args[0], args[1]))
	}

	diffs, err := baur.DiffInputs(inputs1, inputs2)
	exitOnErr(err)

	c.printOutput(diffs)

	if len(diffs) > 0 {
		exitFunc(2)
	}

	exitFunc(0)
}

func (c *diffInputsCmd) getDiffInputArgDetails(repo *baur.Repository, args []string) []*diffInputArgDetails {
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
		tasks := mustArgToTasks(repo, c.gitRepo, mustHaveTasks)

		for _, task := range tasks {
			for _, argDetails := range results {
				if argDetails.runID == "" && task.AppName == argDetails.appName && task.Name == argDetails.taskName {
					argDetails.task = task
				}
			}
		}

		for _, argDetails := range results {
			if argDetails.runID == "" && argDetails.task == nil {
				exitOnErr(fmt.Errorf("%s: task not found", argDetails.arg))
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

func (c *diffInputsCmd) getTaskRunInputs(repo *baur.Repository, argDetails *diffInputArgDetails, withInputStrs bool) (*baur.Inputs, *storage.TaskRunWithID) {
	if argDetails.task != nil {
		var inputStrs []baur.Input
		if withInputStrs {
			inputStrs = baur.AsInputStrings(c.inputStr...)
		}
		inputResolver := baur.NewInputResolver(c.gitRepo, repo.Path, inputStrs, true)

		inputs, err := inputResolver.Resolve(ctx, argDetails.task)
		if err != nil {
			stderr.TaskPrintf(argDetails.task, err.Error())
			os.Exit(1)
		}

		return inputs, nil
	}

	taskRun := getTaskRun(repo, argDetails)

	psql := mustNewCompatibleStorage(repo)
	defer psql.Close()

	storageInputs, err := psql.Inputs(ctx, taskRun.ID)
	exitOnErr(err)

	return toBaurInputs(storageInputs), taskRun
}

func getTaskRun(repo *baur.Repository, argDetails *diffInputArgDetails) *storage.TaskRunWithID {
	psql := mustNewCompatibleStorage(repo)
	defer psql.Close()

	if strings.Contains(argDetails.runID, "^") {
		return getPreviousTaskRun(psql, argDetails)
	}

	id, err := strconv.Atoi(argDetails.runID)
	exitOnErr(err)

	return getTaskRunByID(psql, id)
}

func getPreviousTaskRun(psql storage.Storer, argDetails *diffInputArgDetails) *storage.TaskRunWithID {
	filters := []*storage.Filter{
		{
			Field:    storage.FieldApplicationName,
			Operator: storage.OpEQ,
			Value:    argDetails.appName,
		},
		{
			Field:    storage.FieldTaskName,
			Operator: storage.OpEQ,
			Value:    argDetails.taskName,
		},
	}

	sorters := []*storage.Sorter{
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
		storage.NoLimit,
		func(record *storage.TaskRunWithID) error {
			retrieved++
			if retrieved == runPosition {
				taskRun = record
				return found
			}
			return nil
		},
	)

	if err != nil && errors.Unwrap(err) != found { //nolint:errorlint
		exitOnErr(err)
	}

	if runPosition > retrieved {
		exitOnErr(fmt.Errorf("run %s does not exist, only %d task-run(s) exist(s)", argDetails.arg, retrieved))
	}

	return taskRun
}

func getTaskRunByID(psql storage.Storer, id int) *storage.TaskRunWithID {
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
		storage.NoLimit,
		func(run *storage.TaskRunWithID) error {
			taskRun = run
			return nil
		},
	)

	if errors.Is(err, storage.ErrNotExist) {
		err = fmt.Errorf("task-run %d does not exist", id)
	}
	exitOnErr(err)

	return taskRun
}

func (c *diffInputsCmd) printOutput(diffs []*baur.InputDiff) {
	if !c.quiet {
		var formatter Formatter

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
