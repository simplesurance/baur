package command

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur/v5/internal/command/term"
	"github.com/simplesurance/baur/v5/pkg/storage"
)

const initDbExample = `
baur init db postgres://postgres@localhost:5432/baur?sslmode=disable
`

var initDbLongHelp = fmt.Sprintf(`
Creates the baur tables in a PostgreSQL database.

The Postgres URL is read from the repository configuration file.
Alternatively the URL can be passed as argument or
by setting the '%s' environment variable.`,
	term.Highlight(envVarPSQLURL))

var initDbCmd = &cobra.Command{
	Use:               "db [POSTGRES-URL]",
	Short:             "create baur tables in a PostgreSQL database",
	Example:           strings.TrimSpace(initDbExample),
	Long:              strings.TrimSpace(initDbLongHelp),
	Run:               initDb,
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: cobra.NoFileCompletions,
}

func init() {
	initCmd.AddCommand(initDbCmd)
}

func initDb(_ *cobra.Command, args []string) {
	var dbURL string

	if len(args) == 1 {
		dbURL = args[0]
	} else {
		var err error
		dbURL, err = postgresqlURL()
		exitOnErr(err)
	}

	storageClt, err := newStorageClient(dbURL)
	exitOnErr(err, "establishing connection failed")
	defer storageClt.Close()

	err = storageClt.Init(ctx)
	if errors.Is(err, storage.ErrExists) {
		fatal("database already exists")
	}
	exitOnErr(err)

	stdout.Println("database tables created successfully")
}
