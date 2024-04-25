package command

import (
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur/v3/internal/command/term"
	"github.com/simplesurance/baur/v3/internal/format/csv"
	"github.com/simplesurance/baur/v3/internal/format/table"
	"github.com/simplesurance/baur/v3/pkg/baur"
)

func init() {
	lsCmd.AddCommand(&newLsInputsCmd().Command)
}

type lsInputsCmd struct {
	cobra.Command

	csv        bool
	quiet      bool
	showDigest bool
	inputStr   []string
}

func newLsInputsCmd() *lsInputsCmd {
	cmd := lsInputsCmd{
		Command: cobra.Command{
			Use:   "inputs APP_NAME.TASK_NAME|RUN_ID|APP_DIR",
			Short: "list inputs of a task or task run",
			Args:  cobra.ExactArgs(1),
			ValidArgsFunction: newCompleteTargetFunc(completeTargetFuncOpts{
				withoutWildcards: true,
			}),
		},
	}

	cmd.Run = cmd.run

	cmd.Flags().BoolVar(&cmd.csv, "csv", false,
		"show output in RFC4180 CSV format")

	cmd.Flags().BoolVarP(&cmd.quiet, "quiet", "q", false,
		"Suppress printing headers and the total input digest")

	cmd.Flags().BoolVar(&cmd.showDigest, "digests", false,
		"show digests")

	cmd.Flags().StringArrayVar(&cmd.inputStr, "input-str", nil,
		"include a string as input, can be specified multiple times")

	return &cmd
}

func (c *lsInputsCmd) run(_ *cobra.Command, args []string) {
	var inputs *baur.Inputs

	if taskID, err := strconv.Atoi(args[0]); err == nil {
		if len(c.inputStr) != 0 {
			stderr.Printf("--input-str can only be specified for task-names")
			os.Exit(1)
		}

		inputs = c.mustGetTaskRunInputs(taskID)
	} else {
		inputs = c.mustGetTaskInputs(args[0])
	}

	inputs.Sort()
	c.mustPrintTaskInputs(inputs)
}

func (c *lsInputsCmd) mustGetTaskRunInputs(taskRunID int) *baur.Inputs {
	repo := mustFindRepository()

	storageClt := mustNewCompatibleStorage(repo)
	defer storageClt.Close()

	inputs, err := storageClt.Inputs(ctx, taskRunID)
	exitOnErr(err)

	return toBaurInputs(inputs)
}

func (c *lsInputsCmd) mustGetTaskInputs(taskSpec string) *baur.Inputs {
	repo := mustFindRepository()
	vcsState := mustGetRepoState(repo.Path)
	task := mustArgToTask(repo, vcsState, taskSpec)
	inputResolver := baur.NewInputResolver(
		vcsState,
		repo.Path,
		baur.AsInputStrings(c.inputStr...),
		true,
	)

	inputs, err := inputResolver.Resolve(ctx, task)
	exitOnErr(err)

	return inputs
}

func (c *lsInputsCmd) mustPrintTaskInputs(inputs *baur.Inputs) {
	var formatter Formatter
	var headers []string
	writeHeaders := !c.quiet

	if writeHeaders {
		headers = []string{"Input"}

		if c.showDigest {
			headers = append(headers, "Digest")
		}
	}

	if c.csv {
		formatter = csv.New(headers, stdout)
	} else {
		formatter = table.New(headers, stdout)
	}

	for _, input := range inputs.Inputs() {
		if !c.showDigest {
			mustWriteRow(formatter, input)
			continue
		}

		digest, err := input.Digest()
		exitOnErrf(err, "%s: calculating digest failed", input)

		mustWriteRow(formatter, input, digest.String())
	}

	err := formatter.Flush()
	exitOnErr(err)

	if c.showDigest && !c.quiet && !c.csv {
		totalDigest, err := inputs.Digest()
		exitOnErr(err, "calculating total input digest failed")

		stdout.Printf("\nTotal Input Digest: %s\n", term.Highlight(totalDigest.String()))
	}
}
