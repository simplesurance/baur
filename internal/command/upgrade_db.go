package command

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur/v3/internal/command/term"
	"github.com/simplesurance/baur/v3/pkg/baur"
	"github.com/simplesurance/baur/v3/pkg/storage"
)

var upgradeDbLongHelp = fmt.Sprintf(`
Upgrade the database schema.

If the database schema is from an older baur version, the schema is updated.
This changes the database structure and makes the database incompatible with
older baur version.
It is not reversible.

The Postgres URL is read from the repository configuration file.
Alternatively the URL can be passed as argument or
by setting the '%s' environment variable.`,
	term.Highlight(envVarPSQLURL))

func init() {
	upgradeCmd.AddCommand(&newUpgradeDatabaseCmd().Command)
}

type upgradeDbCmd struct {
	cobra.Command
}

func newUpgradeDatabaseCmd() *upgradeDbCmd {
	cmd := upgradeDbCmd{
		Command: cobra.Command{
			Use:               "db [POSTGRES-URL]",
			Short:             "upgrade the database schema",
			Long:              strings.TrimSpace(upgradeDbLongHelp),
			Args:              cobra.MaximumNArgs(1),
			ValidArgsFunction: cobra.NoFileCompletions,
		},
	}

	cmd.Run = cmd.run

	return &cmd
}

func (*upgradeDbCmd) run(_ *cobra.Command, args []string) {
	var dbURL string

	if len(args) == 1 {
		dbURL = args[0]
	} else {
		repo, err := findRepository()
		if err != nil {
			if os.IsNotExist(err) {
				stderr.Printf("could not find '%s' repository config file.\n"+
					"Run '%s' first or pass the Postgres URL as argument.\n",
					term.Highlight(baur.RepositoryCfgFile), term.Highlight(cmdInitRepo))
				exitFunc(1)
			}

			stderr.Println(err)
			exitFunc(1)
		}

		dbURL = mustGetPSQLURI(repo.Cfg)
	}

	clt, err := newStorageClient(dbURL)
	exitOnErr(err, "establishing database connection failed")
	defer clt.Close()

	curVer, err := clt.SchemaVersion(ctx)
	if errors.Is(err, storage.ErrNotExist) {
		fatalf("database not found, run '%s' to create the database",
			term.Highlight("baur init db"),
		)
	}
	exitOnErr(err, "querying database schema version failed")

	if curVer == clt.RequiredSchemaVersion() {
		stdout.Println("database schema is already up to date, nothing to do")
		return
	}

	if curVer > clt.RequiredSchemaVersion() {
		fatal("database schema is from a newer baur version, please update baur")
	}

	err = clt.Upgrade(ctx)
	exitOnErr(err, "upgrading database schema failed")

	stdout.Printf("database schema successfully upgraded from version %d to %d\n", curVer, clt.RequiredSchemaVersion())
}
