package baur

import (
	"fmt"
	"path/filepath"
	"strings"

	"golang.org/x/exp/maps"

	"github.com/simplesurance/baur/v3/internal/fs"
	"github.com/simplesurance/baur/v3/internal/set"
	"github.com/simplesurance/baur/v3/pkg/cfg"
	"github.com/simplesurance/baur/v3/pkg/cfg/resolver"
)

type Logger interface {
	Debugf(format string, v ...any)
}

// Loader discovers and instantiates apps and tasks.
type Loader struct {
	logger          Logger
	includeDB       *cfg.IncludeDB
	repositoryRoot  string
	appConfigPaths  []string
	gitCommitIDFunc func() (string, error)
}

// NewLoader instantiates a Loader.
// When an app config is loaded the DefaultResolvers are applied on the content
// before they are merged with their includes.  The gitCommitIDFunc is used as
// config resolved to resolve $GITCOMMIT variables.
func NewLoader(repoCfg *cfg.Repository, gitCommitIDFunc func() (string, error), logger Logger) (*Loader, error) {
	repositoryRootDir := filepath.Dir(repoCfg.FilePath())

	appConfigPaths, err := findAppConfigs(repositoryRootDir, repoCfg.Discover.Dirs, repoCfg.Discover.SearchDepth, logger)
	if err != nil {
		return nil, fmt.Errorf("discovering application config files failed: %w", err)
	}

	logger.Debugf("loader: found the following application configs:\n%s", strings.Join(appConfigPaths, "\n"))

	return &Loader{
		logger:          logger,
		repositoryRoot:  repositoryRootDir,
		includeDB:       cfg.NewIncludeDB(logger.Debugf),
		appConfigPaths:  appConfigPaths,
		gitCommitIDFunc: gitCommitIDFunc,
	}, nil
}

// LoadTasks loads the tasks of apps that match the passed specifier.
// Specifier format is (<APP-SPEC>[.<TASK-SPEC>])|PATH
// <APP-SPEC> is:
//   - <APP-NAME> or
//   - '*'
//
// <TASK-SPEC> is:
//   - Task Name or
//   - '*'
//
// If no specifier is passed all tasks of all apps are returned.
// If multiple specifiers match the same task, it's only returned 1x in the returned slice.
func (a *Loader) LoadTasks(specifier ...string) ([]*Task, error) {
	var result []*Task

	specs, err := parseSpecs(specifier)
	if err != nil {
		return nil, err
	}

	specs.all = specs.all || len(specifier) == 0

	apps, err := a.apps(specs)
	if err != nil {
		return nil, err
	}

	if specs.all {
		return a.allTasks(apps)
	}

	result, err = a.allTasks(apps)
	if err != nil {
		return nil, err
	}

	tasks, err := a.tasks(specs.taskSpecs)
	if err != nil {
		return nil, err
	}
	result = append(result, tasks...)

	return dedupTasks(result), nil
}

// LoadApps loads the apps that match the passed specifiers.
// Valid specifiers are:
// - application directory path
// - <APP-NAME>
// - '*'
// If no specifier is passed all apps are returned.
// If multiple specifiers match the same app, it's only returned 1x in the returned slice.
func (a *Loader) LoadApps(specifier ...string) ([]*App, error) {
	specs, err := parseSpecs(specifier)
	if err != nil {
		return nil, err
	}

	if len(specs.taskSpecs) > 0 {
		return nil, fmt.Errorf("invalid app specifiers: %s", specs.taskSpecs)
	}

	specs.all = specs.all || len(specifier) == 0

	return a.apps(specs)
}

// appNames discovers and loads the apps with the given names.
// If no names are passed, a nil []*App slice is returned.
func (a *Loader) appNames(names ...string) ([]*App, error) {
	if len(names) == 0 {
		return nil, nil
	}

	namesMap := set.From(names)
	result := make([]*App, 0, len(names))

	a.logger.Debugf("loader: loading the following apps by name: %+v", names)

	for _, path := range a.appConfigPaths {
		if len(namesMap) == 0 {
			return result, nil
		}

		path, err := filepath.Abs(path)
		if err != nil {
			return nil, err
		}

		appCfg, err := cfg.AppFromFile(path)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}

		if !namesMap.Contains(appCfg.Name) {
			continue
		}

		app, err := a.fromCfg(appCfg)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}

		result = append(result, app)

		delete(namesMap, appCfg.Name)
	}

	notFoundApps := namesMap.Slice()
	if len(notFoundApps) != 0 {
		return nil, fmt.Errorf("could not find the following apps: %s", strings.Join(notFoundApps, ", "))
	}

	return result, nil
}

func (a *Loader) allApps() ([]*App, error) {
	a.logger.Debugf("loader: loading all apps")

	result := make([]*App, 0, len(a.appConfigPaths))

	for _, path := range a.appConfigPaths {
		app, err := a.appPath(path)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}

		result = append(result, app)
	}

	return result, nil
}

func taskCount(apps []*App) int {
	var cnt int
	for _, app := range apps {
		cnt += len(app.cfg.Tasks)
	}
	return cnt
}

func (a *Loader) allTasks(apps []*App) ([]*Task, error) {
	taskCnt := taskCount(apps)
	tasks := make(map[string]*Task, taskCnt)
	tasksWithTaskInfoInputs := make([]*Task, 0, taskCnt)

	for _, app := range apps {
		for _, taskCfg := range app.cfg.Tasks {
			task := NewTask(taskCfg, app.Name, app.repositoryRootPath, app.Path)
			tasks[task.ID] = task
			if len(task.UnresolvedInputs.TaskInfos) > 0 {
				tasksWithTaskInfoInputs = append(tasksWithTaskInfoInputs, task)
			}
		}
	}

	for _, task := range tasksWithTaskInfoInputs {
		if err := task.setTaskInfoDependencies(tasks); err != nil {
			return nil, err
		}
	}

	return maps.Values(tasks), nil
}

// appDirs load apps from the given directories.
func (a *Loader) appDirs(dirs ...string) ([]*App, error) {
	result := make([]*App, 0, len(dirs))

	for _, dir := range dirs {
		cfgPath := filepath.Join(dir, AppCfgFile)

		app, err := a.appPath(cfgPath)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", cfgPath, err)
		}

		result = append(result, app)
	}

	return result, nil
}

// appPath loads the app from the config file.
func (a *Loader) appPath(appConfigPath string) (*App, error) {
	a.logger.Debugf("loader: loading app from %q", appConfigPath)

	appConfigPath, err := filepath.Abs(appConfigPath)
	if err != nil {
		return nil, err
	}

	appCfg, err := cfg.AppFromFile(appConfigPath)
	if err != nil {
		return nil, err
	}

	return a.fromCfg(appCfg)
}

func (a *Loader) appTaskByName(app *App, taskName string) (*Task, error) {
	// TODO: make this more efficient, all tasks of an app are instantiated and then only a single one of those is returned.
	// Instantiate only needed one instead.
	tasks, err := a.allTasks([]*App{app})
	if err != nil {
		return nil, err
	}

	for _, task := range tasks {
		if task.Name == taskName {
			return task, nil
		}
	}

	return nil, nil
}

// tasks load all tasks for the given taskSpecs.
// wildcards are only supported for appNames.
func (a *Loader) tasks(taskSpecs []*taskSpec) ([]*Task, error) {
	result := make([]*Task, 0, len(taskSpecs))
	taskSpecMap := make(map[string][]string, len(taskSpecs))
	appNames := make([]string, 0, len(taskSpecs))

	for _, t := range taskSpecs {
		val, exist := taskSpecMap[t.appName]
		if exist {
			taskSpecMap[t.appName] = append(val, t.taskName)
			continue
		}

		appNames = append(appNames, t.appName)
		taskSpecMap[t.appName] = []string{t.taskName}
	}

	var apps []*App
	var err error
	if _, exist := taskSpecMap["*"]; exist {
		apps, err = a.allApps()
	} else {
		apps, err = a.appNames(appNames...)
	}
	if err != nil {
		return nil, err
	}

	for _, app := range apps {
		for _, spec := range taskSpecMap[app.Name] {
			task, err := a.appTaskByName(app, spec)
			if err != nil {
				return nil, err
			}

			if task == nil {
				return nil, fmt.Errorf("app %q has no task %q", app, spec)
			}

			result = append(result, task)
		}

		// taskSpecs that match all apps are optional,
		// e.g. it's ok if **not** all apps have a task called "check"
		for _, spec := range taskSpecMap["*"] {
			task, err := a.appTaskByName(app, spec)
			if err != nil {
				return nil, err
			}

			if task != nil {
				result = append(result, task)
			}
		}
	}

	return result, nil
}

func (a *Loader) apps(specs *specs) ([]*App, error) {
	if specs.all {
		return a.allApps()
	}

	result := make([]*App, 0, len(specs.appDirs)+len(specs.appNames))

	for _, path := range specs.appDirs {
		apps, err := a.appDirs(path)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}

		result = append(result, apps...)
	}

	apps, err := a.appNames(specs.appNames...)
	if err != nil {
		return nil, err
	}

	result = append(result, apps...)

	return dedupApps(result), nil
}

func (a *Loader) fromCfg(appCfg *cfg.App) (*App, error) {
	resolvers := resolver.NewGoTemplate(appCfg.Name, a.repositoryRoot, a.gitCommitIDFunc)

	err := appCfg.Merge(a.includeDB, resolvers)
	if err != nil {
		return nil, fmt.Errorf("merging includes failed: %w", err)
	}

	err = appCfg.Resolve(resolvers)
	if err != nil {
		return nil, fmt.Errorf("resolving variables in config failed: %w", err)
	}

	err = appCfg.Validate()
	if err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	app, err := NewApp(appCfg, a.repositoryRoot)
	if err != nil {
		return nil, err
	}

	return app, nil
}

// IsAppDirectory returns true if the directory contains an app config file.
func isAppDirectory(dir string) bool {
	cfgPath := filepath.Join(dir, AppCfgFile)
	isFile, _ := fs.IsFile(cfgPath)

	return isFile
}

func findAppConfigs(repoDir string, searchDirs []string, searchDepth int, logger Logger) ([]string, error) {
	appDirs := set.Set[string]{}
	visitedSearchDirs := make(map[string]struct{}, len(searchDirs))

	for _, searchDir := range searchDirs {
		realSearchDir, err := fs.RealPath(fs.AbsPath(repoDir, searchDir))
		if err != nil {
			return nil, err
		}

		if _, alreadyVisited := visitedSearchDirs[realSearchDir]; alreadyVisited {
			logger.Debugf("loader: multiple entries (%q and another) in application_dirs list in %q resolve to the same realpath: %q",
				searchDir, RepositoryCfgFile, realSearchDir)
			continue
		}
		visitedSearchDirs[realSearchDir] = struct{}{}

		if err := fs.DirsExist(realSearchDir); err != nil {
			return nil, fmt.Errorf("application search directory: %w", err)
		}

		cfgPaths, err := fs.FindFilesInSubDir(realSearchDir, AppCfgFile, searchDepth)
		if err != nil {
			return nil, err
		}

		for _, path := range cfgPaths {
			if appDirs.Contains(path) {
				logger.Debugf("loader: multiple entries in application_dirs list in %q discover the same app cfg %q,"+
					"remove duplicate entries to improve performance", RepositoryCfgFile, path)
				continue
			}
			appDirs.Add(path)
		}
	}

	return appDirs.Slice(), nil
}

func dedupApps(apps []*App) []*App {
	return set.From(apps).Slice()
}

func dedupTasks(tasks []*Task) []*Task {
	return set.From(tasks).Slice()
}
