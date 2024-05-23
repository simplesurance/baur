package command

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/simplesurance/baur/v3/internal/command/flag"
	"github.com/simplesurance/baur/v3/internal/command/term"
	"github.com/simplesurance/baur/v3/internal/format/csv"
	"github.com/simplesurance/baur/v3/internal/format/json"
	"github.com/simplesurance/baur/v3/internal/format/table"
	"github.com/simplesurance/baur/v3/internal/log"
	"github.com/simplesurance/baur/v3/internal/prettyprint"
	"github.com/simplesurance/baur/v3/internal/vcs/git"
	"github.com/simplesurance/baur/v3/pkg/baur"
	"github.com/simplesurance/baur/v3/pkg/cfg"
	"github.com/simplesurance/baur/v3/pkg/storage"
	"github.com/simplesurance/baur/v3/pkg/storage/postgres"
)

type Formatter interface {
	WriteRow(Row ...any) error
	Flush() error
}

var ErrPSQLURIMissing = errors.New(
	"PostgreSQL connection information is missing.\n" +
		"- set postgres_url in your repository config or\n" +
		"- set the $" + envVarPSQLURL + "environment variable",
)

var targetHelp = fmt.Sprintf(`%s is in the format %s
Examples:
- 'shop' matches all tasks of the app named shop
- 'shop.*' or 'shop' matches all tasks of the app named shop
- '*.build' matches tasks named build of all applications
- '*.*' matches all tasks of all applications`,
	term.Highlight("TARGET"),
	term.Highlight("(APP_NAME|*)[.TASK_NAME|*]"),
)

// envVarPSQLURL contains the name of an environment variable in that the
// postgresql URI can be stored
const envVarPSQLURL = "BAUR_POSTGRESQL_URL"

func findRepository() (*baur.Repository, error) {
	if repositoryPath != "" {
		cfgPath := filepath.Join(repositoryPath, baur.RepositoryCfgFile)
		log.Debugf("loading repository config: %q\n", cfgPath)
		repo, err := baur.NewRepository(cfgPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil, fmt.Errorf("baur repository not found, ensure %q exists", cfgPath)
			}

			return nil, err
		}

		return repo, nil
	}

	log.Debugln("searching for repository config...")
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	path, err := baur.FindRepositoryCfg(cwd)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fatalf("baur repository not found, ensure a %q file exist in the current or a parent directory\n",
				baur.RepositoryCfgFile)
		}
		return nil, err
	}

	log.Debugf("repository config found: %q", path)
	return baur.NewRepository(path)
}

func mustFindRepository() *baur.Repository {
	repo, err := findRepository()
	if err != nil {
		exitOnErr(err)
	}

	return repo
}

func mustArgToTask(repo *baur.Repository, gitRepo *git.Repository, arg string) *baur.Task {
	tasks := mustArgToTasks(repo, gitRepo, []string{arg})
	if len(tasks) > 1 {
		stderr.Printf("argument %q matches multiple tasks, must match only 1 task\n", arg)
		exitFunc(1)
	}

	// mustArgToApps ensures that >=1 apps are returned
	return tasks[0]
}

func mustArgToApp(repo *baur.Repository, arg string) *baur.App {
	apps := mustArgToApps(repo, []string{arg})
	if len(apps) > 1 {
		stderr.Printf("argument %q matches multiple apps, must match only 1 app\n", arg)
		exitFunc(1)
	}

	// mustArgToApps ensures that >=1 apps are returned
	return apps[0]
}

// newStorageClient creates a new postgresql storage client.
// If the environment variable BAUR_PSQL_URI is set, this uri is used instead
// of the configuration specified in the baur.Repository object
func newStorageClient(psqlURI string) (storage.Storer, error) {
	uri := psqlURI

	if envURI := getPSQLURIEnv(); envURI != "" {
		uri = envURI
	}

	var logger postgres.Logger
	if verboseFlag {
		logger = log.StdLogger
	}

	return postgres.New(ctx, uri, logger)
}

// mustGetPSQLURI returns if it's set the URI from the environment variable
// envVarPSQLURL, otherwise if it's set the psql uri from the repository config,
// if it's also not empty prints an error and exits.
func mustGetPSQLURI(cfg *cfg.Repository) string {
	uri := getPSQLURI(cfg)
	if uri == "" {
		exitOnErr(ErrPSQLURIMissing)
	}

	return uri
}

func getPSQLURI(cfg *cfg.Repository) string {
	if uri := getPSQLURIEnv(); uri != "" {
		return uri
	}

	return cfg.Database.PGSQLURL
}

func getPSQLURIEnv() string {
	if envURI := os.Getenv(envVarPSQLURL); len(envURI) != 0 {
		log.Debugf("using postgresql connection URL from $%s environment variable",
			envVarPSQLURL)

		return envURI
	}

	log.Debugf("environment variable $%s not set", envVarPSQLURL)
	return ""
}

// postgresqlURL returns the value of the environment variable [envVarPSQLURL],
// if is set.
// Otherwise it searches for a baur repository and returns the postgresql url
// from the repository config.
// If the repository object is needed, use [mustNewCompatibleStorage]
// instead, to prevent that the repository is discovered + it's config parsed
// multiple times.
func postgresqlURL() (string, error) {
	if url := os.Getenv(envVarPSQLURL); url != "" {
		return url, nil
	}

	repo, err := findRepository()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("can not locate postgresql database\n"+
				"- the environment variable $%s is not set\n"+
				"- a baur repository was not found: %s", envVarPSQLURL, err,
			)
		}
		return "", err
	}

	if repo.Cfg.Database.PGSQLURL == "" {
		return "", ErrPSQLURIMissing
	}

	return repo.Cfg.Database.PGSQLURL, nil
}

func mustNewCompatibleStorageRepo(r *baur.Repository) storage.Storer {
	return mustNewCompatibleStorage(mustGetPSQLURI(r.Cfg))
}

func mustNewCompatibleStorage(uri string) storage.Storer {
	clt, err := newStorageClient(uri)
	exitOnErr(err, "creating postgresql storage client failed")

	if err := clt.IsCompatible(ctx); err != nil {
		if errors.Is(err, storage.ErrNotExist) {
			fatal("baur postgresql database not found\n" +
				" - ensure that the postgresql URL is correct,\n" +
				" - run 'baur init db' to create the database and schema")

		}
		clt.Close()
		exitOnErr(err)
	}

	return clt
}

func mustGetRepoState(dir string) *git.Repository {
	repo, err := git.NewRepositoryWithCheck(dir)
	if err != nil {
		if errors.Is(err, git.ErrRepositoryNotFound) {
			fatalf("git repository not found, the baur repository (%s) must be part of a git repository", dir)
		}

		exitOnErr(err)
	}

	return repo
}

func mustArgToTasks(repo *baur.Repository, vcs *git.Repository, args []string) []*baur.Task {
	appLoader, err := baur.NewLoader(repo.Cfg, vcs.CommitID, log.StdLogger)
	exitOnErr(err)

	tasks, err := appLoader.LoadTasks(args...)
	exitOnErr(err)

	if len(tasks) == 0 {
		fatalf("could not find any tasks\n"+
			"- ensure the [Discover] section is correct in %s\n"+
			"- ensure that you have >1 application dirs "+
			"containing a %s file with task definitions",
			repo.CfgPath, baur.AppCfgFile)
	}

	return tasks
}

func argToApps(repo *baur.Repository, args []string) ([]*baur.App, error) {
	var apps []*baur.App

	repoState := mustGetRepoState(repo.Path)

	appLoader, err := baur.NewLoader(repo.Cfg, repoState.CommitID, log.StdLogger)
	if err != nil {
		return nil, err
	}

	apps, err = appLoader.LoadApps(args...)
	if err != nil {
		return nil, err
	}

	if len(apps) == 0 {
		return nil, fmt.Errorf("could not find any applications\n"+
			"- ensure the [Discover] section is correct in %s\n"+
			"- ensure that you have >1 application dirs "+
			"containing a %s file",
			repo.CfgPath, baur.AppCfgFile)
	}

	return apps, nil
}

func mustArgToApps(repo *baur.Repository, args []string) []*baur.App {
	apps, err := argToApps(repo, args)
	exitOnErr(err)

	return apps
}

func mustWriteRow(fmt Formatter, row ...any) {
	err := fmt.WriteRow(row...)
	exitOnErr(err)
}

func exitOnErrf(err error, format string, v ...any) {
	if err == nil {
		return
	}

	stderr.ErrPrintf(err, format, v...)
	exitFunc(1)
}

func fatal(msg ...any) {
	stderr.PrintErrln(msg...)
	exitFunc(1)
}

func fatalf(format string, v ...any) {
	stderr.PrintErrf(format, v...)
	exitFunc(1)
}

func exitOnErr(err error, msg ...any) {
	if err == nil {
		return
	}

	stderr.ErrPrintln(err, msg...)
	exitFunc(1)
}

func mustTaskRepoRelPath(repositoryDir string, task *baur.Task) string {
	path, err := filepath.Rel(repositoryDir, task.Directory)
	exitOnErr(err)

	return path
}

func subStr(input string, start int, length int) string {
	asRunes := []rune(input)

	if start >= len(asRunes) {
		return ""
	}

	if start+length > len(asRunes) {
		length = len(asRunes) - start
	}

	return string(asRunes[start : start+length])
}

func mustUntrackedFilesNotExist(requireCleanGitWorktree bool, gitRepo *git.Repository) {
	if !requireCleanGitWorktree {
		return
	}

	if gitRepo.Name() != git.Name {
		fatalf("--%s was specified but baur repository is not a git repository", flagNameRequireCleanGitWorktree)
	}

	untracked, err := gitRepo.UntrackedFiles()
	exitOnErr(err)
	if len(untracked) != 0 {
		fatal(untrackedFilesExistErrMsg(untracked))
	}
}

func untrackedFilesExistErrMsg(untrackedFiles []string) string {
	return fmt.Sprintf("%s was specified, expecting only tracked unmodified files but found the following untracked or modified files:\n%s",
		term.Highlight("--"+flagNameRequireCleanGitWorktree), term.Highlight(prettyprint.TruncatedStrSlice(untrackedFiles, 10)))
}

func mustNewFormatter(formatterName string, hdrs []string) Formatter {
	switch formatterName {
	case flag.FormatCSV:
		return csv.New(hdrs, stdout)
	case flag.FormatPlain:
		return table.New(hdrs, stdout)
	case flag.FormatJSON:
		return json.New(hdrs, stdout)
	default:
		panic(fmt.Sprintf("BUG: newFormatter: unsupported formatter name: %q", formatterName))
	}
}
