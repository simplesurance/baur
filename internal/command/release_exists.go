package command

import (
	"errors"
	"fmt"
	"strings"

	"github.com/simplesurance/baur/v3/internal/command/term"
	"github.com/simplesurance/baur/v3/pkg/baur"
	"github.com/simplesurance/baur/v3/pkg/storage"

	"github.com/spf13/cobra"
)

var releaseExistsLongHelp = fmt.Sprintf(`
Check if a release with a given name exists.

The command can be run without access to the baur repository, by specifying
the PostgreSQL URI via the environment variable %s.

Exit Codes:
  0 - Success, release exists
  1 - Error
  %d - Release does not exist
  `,
	term.Highlight(envVarPSQLURL),
	exitCodeNotExist,
)

type releaseExistsCmd struct {
	cobra.Command
	quiet bool
}

func init() {
	releaseCmd.AddCommand(&newReleaseExistsCmd().Command)
}

func newReleaseExistsCmd() *releaseExistsCmd {
	cmd := releaseExistsCmd{
		Command: cobra.Command{
			Use:               "exists NAME",
			Short:             "check if a release exists",
			Long:              strings.TrimSpace(releaseExistsLongHelp),
			Args:              cobra.ExactArgs(1),
			ValidArgsFunction: nil, //FIXME: implement completion
		},
	}

	cmd.Run = cmd.run
	cmd.Flags().BoolVarP(
		&cmd.quiet,
		"quiet", "q",
		false,
		"suppress printing a result message",
	)

	return &cmd
}

func (c *releaseExistsCmd) run(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()
	psqlURL, err := postgresqlURL()
	exitOnErr(err)

	storageClt := mustNewCompatibleStorage(psqlURL)

	_, err = baur.ReleaseFromStorage(ctx, storageClt, args[0])
	if errors.Is(err, storage.ErrNotExist) {
		if !c.quiet {
			stdout.Printf("release %s does not exist\n", term.Highlight(args[0]))
		}
		exitFunc(exitCodeNotExist)
	}
	exitOnErr(err)

	if !c.quiet {
		stdout.Printf("release %s exists\n", term.Highlight(args[0]))
	}

}
