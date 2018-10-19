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

type inputsShowConf struct {
	quiet      bool
	showDigest bool
	csv        bool
}

var inputsShowCmd = &cobra.Command{
	Use:   "show [<APP-NAME>|<PATH>]",
	Short: "show resolved build inputs of an application",
	Run:   inputsShow,
	Args:  cobra.ExactArgs(1),
}

var inputsShowConfig inputsShowConf

func init() {
	inputsShowCmd.Flags().BoolVar(&inputsShowConfig.csv, "csv", false,
		"Show output in RFC4180 CSV format")

	inputsShowCmd.Flags().BoolVarP(&inputsShowConfig.quiet, "quiet", "q", false,
		"Only show filepaths")

	inputsShowCmd.Flags().BoolVar(&inputsShowConfig.showDigest, "digests", false,
		"show digests")

	inputsCmd.AddCommand(inputsShowCmd)
}
func inputsShow(cmd *cobra.Command, args []string) {
	var formatter format.Formatter
	var headers []string

	rep := MustFindRepository()
	app := mustArgToApp(rep, args[0])
	writeHeaders := !inputsShowConfig.quiet && !inputsShowConfig.csv

	if len(app.BuildInputPaths) == 0 {
		log.Fatalf("No build inputs are configured in %s of %s", baur.AppCfgFile, app.Name)
	}

	if writeHeaders {
		headers = []string{"Path"}

		if inputsShowConfig.showDigest {
			headers = append(headers, "Digest")
		}
	}

	if inputsShowConfig.csv {
		formatter = csv.New(headers, os.Stdout)
	} else {
		formatter = table.New(headers, os.Stdout)
	}

	inputs, err := app.BuildInputs()
	if err != nil {
		log.Fatalln(err)
	}

	sort.Slice(inputs, func(i, j int) bool {
		return inputs[i].URL() < inputs[j].URL()
	})

	for _, input := range inputs {
		if !inputsShowConfig.showDigest || inputsShowConfig.quiet {
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

	if inputsShowConfig.showDigest && !inputsShowConfig.quiet && !inputsShowConfig.csv {
		totalDigest, err := app.TotalInputDigest()
		if err != nil {
			log.Fatalln("calculating total input digest failed:", err)
		}
		fmt.Printf("\nTotal Build Input Digest: %s\n", highlight(totalDigest.String()))
	}
}
