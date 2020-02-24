package command

import (
	"fmt"
	"os"
	"time"

	"github.com/fatih/color"

	"github.com/simplesurance/baur"
	"github.com/simplesurance/baur/format"
	"github.com/simplesurance/baur/git"
	"github.com/simplesurance/baur/log"
	"github.com/simplesurance/baur/storage"
	"github.com/simplesurance/baur/storage/postgres"
)

// envVarPSQLURL contains the name of an environment variable in that the
// postgresql URI can be stored
const envVarPSQLURL = "BAUR_POSTGRESQL_URL"

var (
	greenHighlight  = color.New(color.FgGreen).SprintFunc()
	redHighlight    = color.New(color.FgRed).SprintFunc()
	yellowHighlight = color.New(color.FgYellow).SprintFunc()
	underline       = color.New(color.Underline).SprintFunc()
	// highlight is a function that highlights parts of strings in the cli output
	highlight = greenHighlight
)

func findRepository() (*baur.Repository, error) {
	log.Debugln("searching for repository root...")

	repo, err := baur.FindRepositoryCwd()
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
		log.Fatalln(err)
	}

	return repo
}

func mustArgToApp(repo *baur.Repository, arg string) *baur.App {
	apps := mustArgToApps(repo, []string{arg})
	if len(apps) > 1 {
		log.Fatalf("argument %q matches multiple apps, must match only 1 app\n", arg)
	}

	// mustArgToApps ensures that >=1 apps are returned
	return apps[0]
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
	var apps []*baur.App

	repoState := git.NewRepositoryState(repo.Path)

	appLoader, err := baur.NewAppLoader(repo.Cfg, repoState.CommitID, log.StdLogger)
	if err != nil {
		log.Fatalln(err)
	}

	if len(args) == 0 {
		apps, err = appLoader.All()
	} else {
		apps, err = appLoader.Load(args...)
	}

	if err != nil {
		log.Fatalln(err)
	}

	if len(apps) == 0 {
		log.Fatalf("could not find any applications\n"+
			"- ensure the [Discover] section is correct in %s\n"+
			"- ensure that you have >1 application dirs "+
			"containing a %s file",
			repo.CfgPath, baur.AppCfgFile)
	}

	return apps
}

func mustWriteRow(fmt format.Formatter, row []interface{}) {
	err := fmt.WriteRow(row)
	if err != nil {
		log.Fatalln(err)
	}
}

func coloredBuildStatus(status baur.BuildStatus) string {
	switch status {
	case baur.BuildStatusInputsUndefined:
		return yellowHighlight(status.String())
	case baur.BuildStatusExist:
		return greenHighlight(status.String())
	case baur.BuildStatusPending:
		return redHighlight(status.String())
	default:
		return status.String()
	}
}

func bytesToMib(bytes int) string {
	return fmt.Sprintf("%.3f", float64(bytes)/1024/1024)
}

func durationToStrSeconds(duration time.Duration) string {
	return fmt.Sprintf("%.3f", duration.Seconds())
}
