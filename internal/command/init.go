package command

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur/v2/internal/command/term"
)

const (
	cmdInitApp      = "baur init app"
	cmdInitBashComp = "baur init bashcomp"
	cmdInitDb       = "baur init db"
	cmdInitRepo     = "baur init repo"
)

var initLongHelp = fmt.Sprintf(`
The init commands initialize baur configuration files,
create baur tables in the database or install bash completion files.

To setup baur for the first time, the following commands should be run:
1.) %s
2.) %s
Optional: %s

Afterwards application configuration files can be created with the
'%s' command.
`, term.Highlight(cmdInitRepo),
	term.Highlight(cmdInitDb),
	term.Highlight(cmdInitBashComp),
	term.Highlight(cmdInitApp))

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "initialize configuration files, the baur database, bashcompletion",
	Long:  strings.TrimSpace(initLongHelp),
}

func init() {
	rootCmd.AddCommand(initCmd)
}
