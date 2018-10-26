package command

import (
	"fmt"
	"os"
	"path"

	"github.com/fatih/color"

	"github.com/simplesurance/baur"
	"github.com/simplesurance/baur/format"
	"github.com/simplesurance/baur/log"
	"github.com/simplesurance/baur/storage"
	"github.com/simplesurance/baur/storage/postgres"
)

// envVarPSQLURL contains the name of an environment variable in that the
// postgresql URI can be stored
const envVarPSQLURL = "BAUR_POSTGRESQL_URL"

var (
	greenHighlight = color.New(color.FgGreen).SprintFunc()
	underline      = color.New(color.Underline).SprintFunc()
	// highlight is a function that highlights parts of strings in the cli output
	highlight = greenHighlight
)

func findRepository() (*baur.Repository, error) {
	log.Debugln("searching for repository root...")

	repo, err := baur.FindRepository()
	if err != nil {
		return nil, err
	}

	log.Debugf("repository root found: %s", repo.Path)

	return repo, nil
}

// MustFindRepository must find repo
func MustFindRepository() *baur.Repository {
	repo, err := findRepository()
	if err != nil {
		if os.IsNotExist(err) {
			log.Fatalf("could not find repository root config file "+
				"ensure the file '%s' exist in the root",
				baur.RepositoryCfgFile)
		}

		log.Fatalln(err)
	}

	return repo
}

func isAppDir(arg string) bool {
	cfgPath := path.Join(arg, baur.AppCfgFile)
	_, err := os.Stat(cfgPath)
	if err == nil {
		return true
	}

	return false
}

func mustArgToApp(repo *baur.Repository, arg string) *baur.App {
	if isAppDir(arg) {
		app, err := repo.AppByDir(arg)
		if err != nil {
			log.Fatalf("could not find application in dir '%s': %s", arg, err)
		}

		return app
	}

	app, err := repo.AppByName(arg)
	if err != nil {
		if os.IsNotExist(err) {
			log.Fatalf("could not find application with name '%s'", arg)
		}
		log.Fatalln(err)
	}

	return app
}

// getPostgresCltWithEnv returns a new postresql storage client,
// if the environment variable BAUR_PSQL_URI is set, this uri is used instead of
// the configuration specified in the baur.Repository object
func getPostgresCltWithEnv(psqlURI string) (*postgres.Client, error) {
	uri := psqlURI

	if envURI := os.Getenv(envVarPSQLURL); len(envURI) != 0 {
		log.Debugf("using postgresql connection URL from $%s environment variable",
			envVarPSQLURL)

		uri = envURI
	} else {
		log.Debugf("environment variable $%s not set", envVarPSQLURL)
	}

	return postgres.New(uri)
}

//mustHavePSQLURI calls log.Fatalf if neither envVarPSQLURL nor the postgres_url
//in the repository config is set
func mustHavePSQLURI(r *baur.Repository) {
	if len(r.PSQLURL) != 0 {
		return
	}

	if len(os.Getenv(envVarPSQLURL)) == 0 {
		log.Fatalf("PostgreSQL connection information is missing.\n"+
			"- set postgres_url in your repository config or\n"+
			"- set the $%s environment variable", envVarPSQLURL)
	}
}

// MustGetPostgresClt must return the PG client
func MustGetPostgresClt(r *baur.Repository) *postgres.Client {
	mustHavePSQLURI(r)

	clt, err := getPostgresCltWithEnv(r.PSQLURL)
	if err != nil {
		log.Fatalf("could not establish connection to postgreSQL db: %s", err)
	}

	return clt
}

func mustGetCommitID(r *baur.Repository) string {
	commitID, err := r.GitCommitID()
	if err != nil {
		log.Fatalln(err)
	}

	return commitID
}

func mustGetGitWorktreeIsDirty(r *baur.Repository) bool {
	isDirty, err := r.GitWorkTreeIsDirty()
	if err != nil {
		log.Fatalln(err)
	}

	return isDirty
}

func vcsStr(v *storage.VCSState) string {
	if len(v.CommitID) == 0 {
		return ""
	}

	if v.IsDirty {
		return fmt.Sprintf("%s-dirty", v.CommitID)
	}

	return v.CommitID
}

func mustArgToApps(repo *baur.Repository, args []string) []*baur.App {
	if len(args) == 0 {
		apps, err := repo.FindApps()
		if err != nil {
			log.Fatalln(err)
		}

		if len(apps) == 0 {
			log.Fatalf("could not find any applications"+
				"- ensure the [Discover] section is correct in %s\n"+
				"- ensure that you have >1 application dirs "+
				"containing a %s file\n",
				repo.CfgPath, baur.AppCfgFile)
		}

		return apps
	}

	apps := make([]*baur.App, 0, len(args))
	for _, arg := range args {
		apps = append(apps, mustArgToApp(repo, arg))
	}

	return apps
}

func mustWriteRow(fmt format.Formatter, row []interface{}) {
	err := fmt.WriteRow(row)
	if err != nil {
		log.Fatalln(err)
	}
}
