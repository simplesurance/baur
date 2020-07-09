package command

import (
	"os"
	"path"
	"strings"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur"
	"github.com/simplesurance/baur/cfg"
	"github.com/simplesurance/baur/internal/command/term"
	"github.com/simplesurance/baur/log"
)

func init() {
	initCmd.AddCommand(initRepoCmd)
}

const initRepoLongHelp = `
Create a new repository configuration file.
This is the first command that should be run when setting up baur for a new repository.
If no argument is passed, the file is created in the current directory.
`

var initRepoCmd = &cobra.Command{
	Use:   "repo [DIR]",
	Short: "create a repository config file",
	Long:  strings.TrimSpace(initRepoLongHelp),
	Run:   initRepo,
	Args:  cobra.MaximumNArgs(1),
}

func initRepo(cmd *cobra.Command, args []string) {
	var repoDir string
	var err error

	if len(args) == 1 {
		repoDir = args[0]
	} else {
		repoDir, err = os.Getwd()
		exitOnErr(err)
	}

	repoCfg := cfg.ExampleRepository()
	repoCfgPath := path.Join(repoDir, baur.RepositoryCfgFile)

	err = repoCfg.ToFile(repoCfgPath)
	if err != nil {
		if os.IsExist(err) {
			log.Fatalf("%s already exist\n", repoCfgPath)
		}

		log.Fatalln(err)
	}

	stdout.Printf("Repository configuration was written to %s\n",
		term.Highlight(repoCfgPath))
	stdout.Printf("\nNext Steps:\n"+
		"1. Adapt your '%s' configuration file, ensure the '%s' parameter is correct\n"+
		"2. Run '%s' to create the baur tables in the PostgreSQL database\n"+
		"3. Run '%s' to create application configuration files\n"+
		"Optional: Run '%s' to setup bash completion\n",
		term.Highlight(baur.RepositoryCfgFile),
		term.Highlight("postgresql_url"),
		term.Highlight(cmdInitDb),
		term.Highlight(cmdInitApp),
		term.Highlight(cmdInitBashComp))
}
