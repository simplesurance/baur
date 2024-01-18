package command

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/simplesurance/baur/v3/internal/command/term"
	"github.com/simplesurance/baur/v3/internal/format"
	"github.com/simplesurance/baur/v3/internal/log"
	"github.com/simplesurance/baur/v3/internal/prettyprint"
	"github.com/simplesurance/baur/v3/internal/vcs"
	"github.com/simplesurance/baur/v3/internal/vcs/git"
	"github.com/simplesurance/baur/v3/pkg/baur"
	"github.com/simplesurance/baur/v3/pkg/cfg"
	"github.com/simplesurance/baur/v3/pkg/storage"
	"github.com/simplesurance/baur/v3/pkg/storage/postgres"
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

func mustArgToTask(repo *baur.Repository, vcs vcs.StateFetcher, arg string) *baur.Task {
	tasks := mustArgToTasks(repo, vcs, []string{arg})
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

	return postgres.New(ctx, uri, logger)
}

// mustGetPSQLURI returns if it's set the URI from the environment variable
// envVarPSQLURL, otherwise if it's set the psql uri from the repository config,
// if it's also not empty prints an error and exits.
func mustGetPSQLURI(cfg *cfg.Repository) string {
	uri := getPSQLURI(cfg)
	if uri == "" {
		stderr.Printf("PostgreSQL connection information is missing.\n"+
			"- set postgres_url in your repository config or\n"+
			"- set the $%s environment variable", envVarPSQLURL)
		exitFunc(1)
	}

	return uri
}

func getPSQLURI(cfg *cfg.Repository) string {
	if url := os.Getenv(envVarPSQLURL); url != "" {
		return url
	}

	return cfg.Database.PGSQLURL
}

// mustNewCompatibleStorage initializes a new postgresql storage client.
// The function ensures that the storage is compatible.
func mustNewCompatibleStorage(r *baur.Repository) storage.Storer {
	clt, err := newStorageClient(mustGetPSQLURI(r.Cfg))
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

func mustArgToTasks(repo *baur.Repository, vcs vcs.StateFetcher, args []string) []*baur.Task {
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

func mustWriteRow(fmt format.Formatter, row ...any) {
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
	stderr.Printf(format, v...)
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

func mustUntrackedFilesNotExist(requireCleanGitWorktree bool, vcsState vcs.StateFetcher) {
	if !requireCleanGitWorktree {
		return
	}

	if vcsState.Name() != git.Name {
		fatalf("--%s was specified but baur repository is not a git repository", flagNameRequireCleanGitWorktree)
	}

	untracked, err := vcsState.UntrackedFiles()
	exitOnErr(err)
	if len(untracked) != 0 {
		fatal(untrackedFilesExistErrMsg(untracked))
	}
}

func untrackedFilesExistErrMsg(untrackedFiles []string) string {
	return fmt.Sprintf("%s was specified, expecting only tracked unmodified files but found the following untracked or modified files:\n%s",
		term.Highlight("--"+flagNameRequireCleanGitWorktree), term.Highlight(prettyprint.TruncatedStrSlice(untrackedFiles, 10)))
}
