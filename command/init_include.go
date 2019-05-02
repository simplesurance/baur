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
Create an include config file in the current directory.
If no argument is passed, the file is named +` + defIncludeFilename + `.`

var initIncludeCmd = &cobra.Command{
	Use:   "include [<FILENAME>]",
	Short: "create an include config file",
	Long:  strings.TrimSpace(initIncludeLongHelp),
	Run:   initInclude,
	Args:  cobra.MaximumNArgs(1),
}

func initInclude(cmd *cobra.Command, args []string) {
	// TODO: Warn if an include file is created in a directory that is not listed in the IncludeDirs directive of the .baur.toml file?
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
