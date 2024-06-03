package command

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/simplesurance/baur/v3/internal/command/term"
	"github.com/simplesurance/baur/v3/internal/log"
	"github.com/simplesurance/baur/v3/pkg/baur"
	"github.com/simplesurance/baur/v3/pkg/storage"

	"github.com/spf13/cobra"
)

var releaseShowLongHelp = fmt.Sprintf(`
Display information about a release.
The information is printed to stdout in JSON format.

The command can be run without access to the baur repository, by specifying
the PostgreSQL URI via the environment variable %s.

  1 - Error
  %d - Release does not exist
`,
	term.Highlight(envVarPSQLURL),
	exitCodeNotExist,
)

type releaseShowCmd struct {
	cobra.Command

	metadataFilePath string
}

func init() {
	releaseCmd.AddCommand(&newReleaseShowCmd().Command)
}

func newReleaseShowCmd() *releaseShowCmd {
	cmd := releaseShowCmd{
		Command: cobra.Command{
			Use:               "show NAME",
			Short:             "display information about a release",
			Long:              strings.TrimSpace(releaseShowLongHelp),
			Args:              cobra.ExactArgs(1),
			ValidArgsFunction: cobra.NoFileCompletions,
		},
	}

	cmd.Flags().StringVarP(
		&cmd.metadataFilePath, "metadata", "m", "",
		"write the stored metadata to the given file path,\n"+
			" instead of including it in the JSON output",
	)

	cmd.Run = cmd.run
	return &cmd
}

func (c *releaseShowCmd) run(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()
	releaseName := args[0]
	writeMetadataTofile := c.metadataFilePath != ""

	psqlURL, err := postgresqlURL()
	exitOnErr(err)

	storageClt := mustNewCompatibleStorage(psqlURL)

	rm := baur.NewReleaseManager(&storageClt, nil, log.StdLogger)
	release, err := rm.GetRelease(ctx, releaseName)
	if errors.Is(err, storage.ErrNotExist) {
		stderr.Printf(
			"release %s does not exist\n",
			term.Highlight(args[0]),
		)
		exitFunc(exitCodeNotExist)
	}
	exitOnErr(err)

	if writeMetadataTofile {
		f, err := os.Create(c.metadataFilePath)
		exitOnErrf(err, "creating metadata file %s failed", c.metadataFilePath)

		err = release.WriteMetadata(f)
		exitOnErrf(err, "writing to metadata file %s failed", c.metadataFilePath)

		err = f.Close()
		exitOnErrf(err, "writing to metadata file %s failed", c.metadataFilePath)
	}

	err = release.ToJSON(stdout, writeMetadataTofile)
	exitOnErr(err)
}
