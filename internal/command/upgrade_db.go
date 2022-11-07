package command

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur/v3/internal/command/term"
	"github.com/simplesurance/baur/v3/pkg/storage"
)

var upgradeDbLongHelp = `
Upgrade the database schema.

If the database schema is from an older baur version, the schema is updated.
This changes the database structure and makes the database incompatible with
older baur version.
This is not reversible.
`

func init() {
	upgradeCmd.AddCommand(&newUpgradeDatabaseCmd().Command)
}

type upgradeDbCmd struct {
	cobra.Command
}

func newUpgradeDatabaseCmd() *upgradeDbCmd {
	cmd := upgradeDbCmd{
		Command: cobra.Command{
			Use:   "db",
			Short: "upgrade the database schema",
			Long:  upgradeDbLongHelp,
		},
	}

	cmd.Run = cmd.run

	return &cmd
}

func (*upgradeDbCmd) run(cmd *cobra.Command, _ []string) {
	repo := mustFindRepository()

	clt, err := newStorageClient(mustGetPSQLURI(repo.Cfg))
	exitOnErr(err, "establishing database connection failed")
	defer clt.Close()

	curVer, err := clt.SchemaVersion(ctx)
	if errors.Is(err, storage.ErrNotExist) {
		stderr.ErrPrintln(fmt.Errorf(
			"database not found, run '%s' to create the database",
			term.Highlight("baur init db"),
		))
		exitFunc(1)
	}
	exitOnErr(err, "querying database schema version failed")

	if curVer == clt.RequiredSchemaVersion() {
		stdout.Println("database schema is already up to date, nothing to do")
		return
	}

	if curVer > clt.RequiredSchemaVersion() {
		stderr.Println("database schema is from a newer baur version, please update baur")
		exitFunc(1)
	}

	err = clt.Upgrade(ctx)
	exitOnErr(err, "upgrading database schema failed")

	stdout.Printf("database schema successfully upgraded from version %d to %d\n", curVer, clt.RequiredSchemaVersion())
}