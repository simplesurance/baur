package command

import (
	"os"
	"sort"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur"
	"github.com/simplesurance/baur/format"
	"github.com/simplesurance/baur/format/csv"
	"github.com/simplesurance/baur/format/table"
	"github.com/simplesurance/baur/internal/command/term"
)

type lsInputsConf struct {
	quiet      bool
	showDigest bool
	csv        bool
}

var lsInputsCmd = &cobra.Command{
	Use:   "inputs <APP-NAME>.<TASK-NAME>]",
	Short: "list resolved task inputs of an application",
	Run:   lsInputs,
	Args:  cobra.ExactArgs(1),
}

var lsInputsConfig lsInputsConf

func init() {
	lsInputsCmd.Flags().BoolVar(&lsInputsConfig.csv, "csv", false,
		"Show output in RFC4180 CSV format")

	lsInputsCmd.Flags().BoolVarP(&lsInputsConfig.quiet, "quiet", "q", false,
		"Only show filepaths")

	lsInputsCmd.Flags().BoolVar(&lsInputsConfig.showDigest, "digests", false,
		"show digests")

	lsCmd.AddCommand(lsInputsCmd)
}

func lsInputs(cmd *cobra.Command, args []string) {
	var formatter format.Formatter
	var headers []string

	rep := MustFindRepository()
	task := mustArgToTask(rep, args[0])
	writeHeaders := !lsInputsConfig.quiet && !lsInputsConfig.csv

	if !task.HasInputs() {
		stderr.TaskPrintf(task, "has no inputs configured")
		os.Exit(1)
	}

	if writeHeaders {
		headers = []string{"Path"}

		if lsInputsConfig.showDigest {
			headers = append(headers, "Digest")
		}
	}

	if lsInputsConfig.csv {
		formatter = csv.New(headers, stdout)
	} else {
		formatter = table.New(headers, stdout)
	}

	inputResolver := baur.NewInputResolver()

	inputs, err := inputResolver.Resolve(rep.Path, task)
	exitOnErr(err)

	sort.Slice(inputs.Files, func(i, j int) bool {
		return inputs.Files[i].RepoRelPath() < inputs.Files[j].RepoRelPath()
	})

	for _, input := range inputs.Files {
		if !lsInputsConfig.showDigest || lsInputsConfig.quiet {
			mustWriteRowVa(formatter, input)
			continue
		}

		digest, err := input.Digest()
		exitOnErrf(err, "%s: calculating digest failed", input)

		mustWriteRowVa(formatter, input, digest.String())
	}

	err = formatter.Flush()
	exitOnErr(err)

	if lsInputsConfig.showDigest && !lsInputsConfig.quiet && !lsInputsConfig.csv {
		totalDigest, err := inputs.Digest()
		exitOnErr(err, "calculating total input digest failed")

		stdout.Printf("\nTotal Build Input Digest: %s\n", term.Highlight(totalDigest.String()))
	}
}
