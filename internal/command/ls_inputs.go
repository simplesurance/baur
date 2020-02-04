package command

import (
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur"
	"github.com/simplesurance/baur/format"
	"github.com/simplesurance/baur/format/csv"
	"github.com/simplesurance/baur/format/table"
	"github.com/simplesurance/baur/log"
)

type lsInputsConf struct {
	quiet      bool
	showDigest bool
	csv        bool
}

var lsInputsCmd = &cobra.Command{
	Use:   "inputs [<APP-NAME>|<PATH>]",
	Short: "list resolved build inputs of an application",
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
	app := mustArgToApp(rep, args[0])
	writeHeaders := !lsInputsConfig.quiet && !lsInputsConfig.csv

	if !app.HasBuildInputs() {
		log.Fatalf("No build inputs are configured in %s of %s", baur.AppCfgFile, app.Name)
	}

	if writeHeaders {
		headers = []string{"Path"}

		if lsInputsConfig.showDigest {
			headers = append(headers, "Digest")
		}
	}

	if lsInputsConfig.csv {
		formatter = csv.New(headers, os.Stdout)
	} else {
		formatter = table.New(headers, os.Stdout)
	}

	inputs, err := app.BuildInputs()
	if err != nil {
		log.Fatalln(err)
	}

	sort.Slice(inputs, func(i, j int) bool {
		return inputs[i].RepoRelPath() < inputs[j].RepoRelPath()
	})

	for _, input := range inputs {
		if !lsInputsConfig.showDigest || lsInputsConfig.quiet {
			mustWriteRow(formatter, []interface{}{input})
			continue
		}

		digest, err := input.Digest()
		if err != nil {
			log.Fatalln("calculating digest failed:", err)
		}

		mustWriteRow(formatter, []interface{}{input, digest.String()})
	}

	if err := formatter.Flush(); err != nil {
		log.Fatalln(err)
	}

	if lsInputsConfig.showDigest && !lsInputsConfig.quiet && !lsInputsConfig.csv {
		totalDigest, err := app.TotalInputDigest()
		if err != nil {
			log.Fatalln("calculating total input digest failed:", err)
		}
		fmt.Printf("\nTotal Build Input Digest: %s\n", highlight(totalDigest.String()))
	}
}
