package command

import (
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur/v4/internal/command/term"
	"github.com/simplesurance/baur/v4/pkg/cfg"
)

func init() {
	initCmd.AddCommand(initIncludeCmd)
}

const defIncludeFilename = "includes.toml"

const initIncludeLongHelp = `
Create an include config file.
If no FILENAME argument is passed, the filename will be '` + defIncludeFilename + `'.`

var initIncludeCmd = &cobra.Command{
	Use:   "include [FILENAME]",
	Short: "create an include config file",
	Long:  strings.TrimSpace(initIncludeLongHelp),
	Run:   initInclude,
	Args:  cobra.MaximumNArgs(1),
}

func initInclude(_ *cobra.Command, args []string) {
	var filename string

	if len(args) == 1 {
		filename = args[0]
		if !strings.HasSuffix(filename, ".toml") {
			filename += ".toml"
		}
	} else {
		filename = defIncludeFilename
	}

	cfgInclude := cfg.ExampleInclude()
	err := cfgInclude.ToFile(filename, cfg.ToFileOptCommented())
	if err != nil {
		if os.IsExist(err) {
			stderr.Printf("%s already exist\n", filename)
			exitFunc(exitCodeError)
		}

		stderr.Println(err)
		exitFunc(exitCodeError)
	}

	stdout.Printf("Include configuration file was written to %s\n",
		term.Highlight(filename))
}
