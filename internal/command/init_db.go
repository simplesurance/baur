package command

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur/v1"
	"github.com/simplesurance/baur/v1/internal/command/term"
	"github.com/simplesurance/baur/v1/log"
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
	Use:     "db [POSTGRES-URL]",
	Short:   "create baur tables in a PostgreSQL database",
	Example: strings.TrimSpace(initDbExample),
	Long:    strings.TrimSpace(initDbLongHelp),
	Run:     initDb,
	Args:    cobra.MaximumNArgs(1),
}

func init() {
	initCmd.AddCommand(initDbCmd)
}

func initDb(cmd *cobra.Command, args []string) {
	var dbURL string

	if len(args) == 0 {
		repo, err := findRepository()
		if err != nil {
			if os.IsNotExist(err) {
				log.Fatalf("could not find '%s' repository config file.\n"+
					"Run '%s' first or pass the Postgres URL as argument.",
					term.Highlight(baur.RepositoryCfgFile), term.Highlight(cmdInitRepo))
			}

			log.Fatalln(err)
		}

		dbURL = repo.PSQLURL
	} else {
		dbURL = args[0]
	}

	storageClt, err := newStorageClient(dbURL)
	exitOnErr(err, "establishing connection failed")

	err = storageClt.Init(ctx)
	exitOnErr(err)

	stdout.Println("database tables created successfully")
}
