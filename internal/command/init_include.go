package command

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur/cfg"
	"github.com/simplesurance/baur/log"
)

func init() {
	initCmd.AddCommand(initIncludeCmd)
}

const defIncludeFilename = "includes.toml"

const initIncludeLongHelp = `
Create an include config file.
If no FILENAME argument is passed, the filename will be '` + defIncludeFilename + `'.`

var initIncludeCmd = &cobra.Command{
	Use:   "include [<FILENAME>]",
	Short: "create an include config file",
	Long:  strings.TrimSpace(initIncludeLongHelp),
	Run:   initInclude,
	Args:  cobra.MaximumNArgs(1),
}

func initInclude(cmd *cobra.Command, args []string) {
	var filename string

	if len(args) == 1 {
		filename = args[0]
	} else {
		filename = defIncludeFilename
	}

	cfg := cfg.ExampleInclude()
	err := cfg.IncludeToFile(filename)
	if err != nil {
		if os.IsExist(err) {
			log.Fatalf("%s already exist\n", filename)
		}

		log.Fatalln(err)
	}

	fmt.Printf("Include configuration file was written to %s\n",
		highlight(filename))
}
