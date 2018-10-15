package command

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur"
	"github.com/simplesurance/baur/digest"
	"github.com/simplesurance/baur/digest/sha384"
	"github.com/simplesurance/baur/log"
)

var inputsShowDigest bool

func init() {
	inputsCmd.Flags().BoolVar(&inputsShowDigest, "digest", false,
		"show digests")
	rootCmd.AddCommand(inputsCmd)
}

const inputsLongHelp = `
Shows build input paths of an application.

The paths to the build inputs must be configured in the .app.toml file.
`

var inputsCmd = &cobra.Command{
	Use:   "inputs [<APP-NAME>|<PATH>]",
	Args:  cobra.ExactArgs(1),
	Short: "list resolved build inputs of an application",
	Long:  strings.TrimSpace(inputsLongHelp),
	Run:   inputs,
}

func inputs(cmd *cobra.Command, args []string) {
	var alLFiles []baur.BuildInput
	var inputDigests []*digest.Digest

	rep := MustFindRepository()
	app := mustArgToApp(rep, args[0])

	if len(app.BuildInputPaths) == 0 {
		log.Fatalf("No build inputs have been configured in the %s file of %s\n", baur.AppCfgFile, app.Name)
	}

	alLFiles, err := app.BuildInputs()
	if err != nil {
		log.Fatalln(err)
	}

	sort.Slice(alLFiles, func(i, j int) bool {
		return alLFiles[i].URL() < alLFiles[j].URL()
	})

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, p := range alLFiles {
		if !inputsShowDigest {
			fmt.Fprintf(tw, "%s\n", p)
			continue
		}

		d, err := p.Digest()
		if err != nil {
			log.Fatalln("creating digest failed:", err)
		}

		inputDigests = append(inputDigests, &d)

		fmt.Fprintf(tw, "%s\t%s\n", p, d.String())
	}

	tw.Flush()

	if inputsShowDigest {
		totalDigest, err := sha384.Sum(inputDigests)
		if err != nil {
			log.Fatalln("calculating total input digest failed:", err)
		}

		fmt.Printf("\ntotal digest: %s\n", totalDigest)

	}

}
