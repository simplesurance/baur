package command

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur"
	"github.com/simplesurance/baur/cfg"
	"github.com/simplesurance/baur/fs"
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
	var filename string

	repo := MustFindRepository()

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

	if !fs.PathIsInDirectories(filename, repo.IncludeDirs...) {
		log.Warnf("File is not in the '%s' defined in the %s file.\n"+
			"baur will not be able to find the include file.",
			highlight("include_dirs"),
			highlight(baur.RepositoryCfgFile))
	}
}
