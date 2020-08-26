package command

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"

	"github.com/simplesurance/baur/v1"
	"github.com/simplesurance/baur/v1/internal/format"
	"github.com/simplesurance/baur/v1/internal/log"
	"github.com/simplesurance/baur/v1/internal/vcs"
	"github.com/simplesurance/baur/v1/storage"
	"github.com/simplesurance/baur/v1/storage/postgres"
)

// envVarPSQLURL contains the name of an environment variable in that the
// postgresql URI can be stored
const envVarPSQLURL = "BAUR_POSTGRESQL_URL"

func findRepository() (*baur.Repository, error) {
	log.Debugln("searching for repository config...")
	path, err := baur.FindRepositoryCfgCwd()
	if err != nil {
		return nil, err
	}

	log.Debugf("repository config found: %q", path)

	return baur.NewRepository(path)
}

// mustFindRepository must find repo
func mustFindRepository() *baur.Repository {
	repo, err := findRepository()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Fatalf("baur repository not found, ensure a %q file exist in the current or a parent directory\n",
				baur.RepositoryCfgFile)
		}

		exitOnErr(err, "locating baur repository failed")
	}

	return repo
}

func mustArgToTask(repo *baur.Repository, arg string) *baur.Task {
	tasks := mustArgToTasks(repo, []string{arg})
	if len(tasks) > 1 {
		log.Fatalf("argument %q matches multiple tasks, must match only 1 task\n", arg)
	}

	// mustArgToApps ensures that >=1 apps are returned
	return tasks[0]
}

func mustArgToApp(repo *baur.Repository, arg string) *baur.App {
	apps := mustArgToApps(repo, []string{arg})
	if len(apps) > 1 {
		log.Fatalf("argument %q matches multiple apps, must match only 1 app\n", arg)
	}

	// mustArgToApps ensures that >=1 apps are returned
	return apps[0]
}

// newStorageClient creates a new postgresql storage client.
// If the environment variable BAUR_PSQL_URI is set, this uri is used instead
// of the configuration specified in the baur.Repository object
func newStorageClient(psqlURI string) (storage.Storer, error) {
	uri := psqlURI

	if envURI := os.Getenv(envVarPSQLURL); len(envURI) != 0 {
		log.Debugf("using postgresql connection URL from $%s environment variable",
			envVarPSQLURL)

		uri = envURI
	} else {
		log.Debugf("environment variable $%s not set", envVarPSQLURL)
	}

	var logger postgres.Logger
	if verboseFlag {
		logger = log.StdLogger
	}

	client, err := postgres.New(ctx, uri, logger)
	if err != nil {
		return nil, err
	}
	return client, nil
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

// mustNewCompatibleStorage initializes a new postgresql storage client.
// The function ensures that the storage is compatible.
func mustNewCompatibleStorage(r *baur.Repository) storage.Storer {
	mustHavePSQLURI(r)

	clt, err := newStorageClient(r.PSQLURL)
	exitOnErr(err, "creating postgresql storage client failed")

	if err := clt.IsCompatible(ctx); err != nil {
		clt.Close()
		exitOnErr(err)
	}

	return clt
}

func mustGetRepoState(dir string) vcs.StateFetcher {
	s, err := vcs.GetState(dir, log.Debugf)
	exitOnErr(err, "failed to evaluate if baur repository is in a VCS repository")

	return s
}

func mustArgToTasks(repo *baur.Repository, args []string) []*baur.Task {
	repoState := mustGetRepoState(repo.Path)

	appLoader, err := baur.NewLoader(repo.Cfg, repoState.CommitID, log.StdLogger)
	exitOnErr(err)

	tasks, err := appLoader.LoadTasks(args...)
	exitOnErr(err)

	if len(tasks) == 0 {
		log.Fatalf("could not find any tasks\n"+
			"- ensure the [Discover] section is correct in %s\n"+
			"- ensure that you have >1 application dirs "+
			"containing a %s file with task definitions",
			repo.CfgPath, baur.AppCfgFile)
	}

	return tasks
}

func mustArgToApps(repo *baur.Repository, args []string) []*baur.App {
	var apps []*baur.App

	repoState := mustGetRepoState(repo.Path)

	appLoader, err := baur.NewLoader(repo.Cfg, repoState.CommitID, log.StdLogger)
	exitOnErr(err)

	apps, err = appLoader.LoadApps(args...)
	exitOnErr(err)

	if len(apps) == 0 {
		log.Fatalf("could not find any applications\n"+
			"- ensure the [Discover] section is correct in %s\n"+
			"- ensure that you have >1 application dirs "+
			"containing a %s file",
			repo.CfgPath, baur.AppCfgFile)
	}

	return apps
}

func mustWriteRow(fmt format.Formatter, row ...interface{}) {
	err := fmt.WriteRow(row...)
	exitOnErr(err)
}

var errorPrefix = color.New(color.FgRed).Sprint("ERROR:")

func exitOnErrf(err error, format string, v ...interface{}) {
	exitOnErr(err, fmt.Sprintf(format, v...))
}

func exitOnErr(err error, msg ...interface{}) {
	if err == nil {
		return
	}

	if len(msg) == 0 {
		fmt.Fprintln(os.Stderr, errorPrefix, err)
		exitFunc(1)
	}

	wholeMsg := fmt.Sprint(msg...)
	fmt.Fprintf(os.Stderr, "%s %s: %s\n", errorPrefix, wholeMsg, err)

	exitFunc(1)
}

func mustTaskRepoRelPath(repositoryDir string, task *baur.Task) string {
	path, err := filepath.Rel(repositoryDir, task.Directory)
	exitOnErr(err)

	return path
}
