package command

import (
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

func init() {
	diffCmd.AddCommand(&newDiffInputsCmd().Command)
}

type diffInputsCmd struct {
	cobra.Command

	csv      bool
	quiet    bool
	inputStr string
}

func newDiffInputsCmd() *diffInputsCmd {
	cmd := diffInputsCmd{
		Command: cobra.Command{
			Use:   "inputs <APP-NAME>.<TASK-NAME>|<RUN-ID> <APP-NAME>.<TASK-NAME>|<RUN-ID>",
			Short: "list inputs that differ between two task-runs",
			Long: `if the inputs match exit code 0 is returned, exit code 2 if they are different or exit code 1 if an error occurs.
when outputting the differences, State 'D' indicates the digests do not match, '+' indicates the input is missing from the first argument and '-' indicates the input is missing from the second argument`,
			Args: diffArgs(),
		},
	}

	cmd.Run = cmd.run

	cmd.Flags().BoolVar(&cmd.csv, "csv", false,
		"show output in RFC4180 CSV format")

	cmd.Flags().BoolVarP(&cmd.quiet, "quiet", "q", false,
		"do not show anything, if the inputs match exit code 0 is returned, exit code 2 if they are different or exit code 1 if an error occurs")

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

		containsWildCardPattern := "^\\*\\..+$|^.+\\.\\*$"
		isNumericPattern := "^\\d+$"
		containsTaskPattern := "^.+\\..+$"

		for _, arg := range args {
			containsWildCard, err := regexp.MatchString(containsWildCardPattern, arg)

			if err != nil {
				return err
			}

			if containsWildCard {
				return fmt.Errorf("%s contains a wild card character, wild card characters are not allowed", arg)
			}

			isValid, err := regexp.MatchString(fmt.Sprintf("%s|%s", isNumericPattern, containsTaskPattern), arg)

			if err != nil {
				return err
			}

			if !isValid {
				return fmt.Errorf("%s does not specify a task or task-run ID", arg)
			}
		}

		return nil
	}
}

func (c *diffInputsCmd) run(cmd *cobra.Command, args []string) {
	if args[0] == args[1] {
		exitOnErr(fmt.Errorf("%s and %s refer to the same task-run", args[0], args[1]))
	}

	app1, task1, runID1 := c.parseDiffSpec(args[0])
	inputs1, run1, err := c.getTaskRunInputs(app1, task1, runID1)
	if err != nil {
		exitOnErr(err)
	}

	app2, task2, runID2 := c.parseDiffSpec(args[1])
	inputs2, run2, err := c.getTaskRunInputs(app2, task2, runID2)
	if err != nil {
		exitOnErr(err)
	}

	if run1 != nil && run2 != nil {
		if run1.ID == run2.ID {
			exitOnErr(fmt.Errorf("%s and %s refer to the same task-run", args[0], args[1]))
		}
	}

	if !c.quiet || c.csv {
		c.printOutput(inputs1, inputs2)
	}

	app1Digest := getDigest(inputs1)
	app2Digest := getDigest(inputs2)

	if app1Digest == app2Digest {
		if c.quiet && !c.csv {
			stdout.Printf("the inputs for %s and %s match", args[0], args[1])
		}
		exitFunc(0)
	}
	if c.quiet && !c.csv {
		stdout.Printf("the inputs for %s and %s differ", args[0], args[1])
	}
	exitFunc(2)
}

// parseDiffSpec splits an argument into the app, task and runID components.
// It relies on the fact that the arguments have already been validated
// in the cobra.Command
func (c *diffInputsCmd) parseDiffSpec(s string) (app, task, runID string) {
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

func (c *diffInputsCmd) getTaskRunInputs(app, task, runID string) (*baur.Inputs, *storage.TaskRunWithID, error) {
	repo := mustFindRepository()

	taskRun, err := getTaskRun(repo, app, task, runID)
	if err != nil {
		exitOnErr(err)
	}

	var inputs *baur.Inputs
	if taskRun == nil {
		task := mustArgToTask(repo, fmt.Sprintf("%s.%s", app, task))

		inputResolver := baur.NewInputResolver()

		inputFiles, err := inputResolver.Resolve(ctx, repo.Path, task)
		if err != nil {
			exitOnErr(err)
		}

		inputs = baur.NewInputs(baur.InputAddStrIfNotEmpty(inputFiles, c.inputStr))
	} else {
		psql := mustNewCompatibleStorage(repo)
		storageInputs, err := psql.Inputs(ctx, taskRun.ID)

		if err != nil {
			exitOnErr(err)
		}

		// Convert the inputs from the DB into baur.Input interface implementation
		var baurInputs []baur.Input
		for _, input := range storageInputs {
			baurInputs = append(baurInputs, &storageInput{input})
		}

		inputs = baur.NewInputs(baur.InputAddStrIfNotEmpty(baurInputs, c.inputStr))
	}

	return inputs, taskRun, nil
}

func getTaskRun(repo *baur.Repository, app, task, runID string) (*storage.TaskRunWithID, error) {
	if runID == "" {
		return nil, nil
	}

	if strings.Contains(runID, "^") {
		return getPreviousTaskRun(repo, app, task, runID)
	}

	id, err := strconv.Atoi(runID)
	if err != nil {
		exitOnErr(err)
	}
	return getTaskRunByID(repo, id)
}

func getPreviousTaskRun(repo *baur.Repository, app, task string, position string) (*storage.TaskRunWithID, error) {
	var filters []*storage.Filter

	filters = append(filters, &storage.Filter{
		Field:    storage.FieldApplicationName,
		Operator: storage.OpEQ,
		Value:    app,
	})

	filters = append(filters, &storage.Filter{
		Field:    storage.FieldTaskName,
		Operator: storage.OpEQ,
		Value:    task,
	})

	sorters := getSorters()

	psql := mustNewCompatibleStorage(repo)

	var taskRuns []*storage.TaskRunWithID
	err := psql.TaskRuns(
		ctx,
		filters,
		sorters,
		func(taskRun *storage.TaskRunWithID) error {
			taskRuns = append(taskRuns, taskRun)
			return nil
		},
	)

	if err != nil {
		exitOnErr(err)
	}

	if len(taskRuns) < len(position) {
		exitOnErr(fmt.Errorf("%s.%s%s does not exist, only %d task-run(s) exist(s)", app, task, position, len(taskRuns)))
	}

	taskRun := taskRuns[len(position)-1]

	return taskRun, nil
}

func getTaskRunByID(repo *baur.Repository, id int) (*storage.TaskRunWithID, error) {
	var filters []*storage.Filter
	filters = append(filters, &storage.Filter{
		Field:    storage.FieldID,
		Operator: storage.OpEQ,
		Value:    id,
	})

	sorters := getSorters()

	psql := mustNewCompatibleStorage(repo)

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

	if err != nil {
		if err == storage.ErrNotExist {
			exitOnErr(fmt.Errorf("task-run %d does not exist", id))
		}
		exitOnErr(err)
	}

	return taskRun, nil
}

func getSorters() []*storage.Sorter {
	var sorters []*storage.Sorter

	defaultSorter := storage.Sorter{
		Field: storage.FieldID,
		Order: storage.OrderDesc,
	}

	sorters = append(sorters, &defaultSorter)

	return sorters
}

func getDigest(inputs *baur.Inputs) string {
	digest, err := inputs.Digest()

	if err != nil {
		exitOnErr(err)
	}

	return digest.String()
}

func (c *diffInputsCmd) printOutput(inputs1, inputs2 *baur.Inputs) {
	var formatter format.Formatter
	headers := []string{"State", "Path", "Digest1", "Digest2"}

	if c.csv {
		formatter = csv.New(headers, stdout)
	} else {
		formatter = table.New(headers, stdout)
	}

	diffs, err := baur.DiffInputs(inputs1, inputs2)
	exitOnErr(err)

	for _, diff := range diffs {
		mustWriteRow(formatter, diff.State, diff.Path, diff.Digest1, diff.Digest2)
	}

	err = formatter.Flush()
	exitOnErr(err)
}
