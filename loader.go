package baur

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/simplesurance/baur/v1/cfg"
	"github.com/simplesurance/baur/v1/internal/fs"
)

type Logger interface {
	Debugf(format string, v ...interface{})
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

	appConfigPaths, err := findAppConfigs(fs.AbsPaths(repositoryRootDir, repoCfg.Discover.Dirs), repoCfg.Discover.SearchDepth)
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

// splitSpecifiers splits the specifiers by apps to load by Name or by Path.
// If the specifiers contain a '*' specifier, nil slices are returned and star is true.
func splitSpecifiers(specifiers []string) (names, cfgPaths []string, star bool) {
	for _, spec := range specifiers {
		if spec == "*" {
			return nil, nil, true
		}

		cfgPath, isAppDir := IsAppDirectory(spec)
		if isAppDir {
			cfgPaths = append(cfgPaths, cfgPath)

			continue
		}

		names = append(names, spec)
	}

	return names, cfgPaths, false
}

// LoadTasks loads the tasks of apps that match the passed specifier.
// Specifier format is <APP-SPEC>[.<TASK-SPEC>].
// <APP-SPEC> is:
//   - <APP-NAME> or
//   - '*'
// <TASK-SPEC> is:
//   - Task Name or
//   - '*'
// If no specifier is passed all tasks of all apps are returned.
// If multiple specifiers match the same task, it's only returned 1x in the returned slice.
func (a *Loader) LoadTasks(specifier ...string) ([]*Task, error) {
	var result []*Task

	if len(specifier) == 0 {
		specifier = []string{"*"}
	}

	for _, spec := range specifier {
		spl := strings.Split(spec, ".")

		if len(spl) == 0 {
			// impossible condition
			panic(fmt.Sprintf("strings.Split(\"%s\", \".\") returned empty slice", spec))
		}

		if len(spl) > 2 {
			return nil, fmt.Errorf("specifier: %q contains > 1 dots ", specifier)
		}

		apps, err := a.LoadApps(spl[0])
		if err != nil {
			return nil, err
		}

		// specifier contains only <APP-SPEC>
		if len(spl) == 1 {
			result = append(result, a.taskSpec(apps, "*")...)
			continue
		}

		result = append(result, a.taskSpec(apps, spl[1])...)
	}

	return result, nil
}

// LoadApps loads the apps that match the passed specifiers.
// Valid specifiers are:
// - application directory path
// - <APP-NAME>
// - '*'
// If no specifier is passed all apps are returned.
// If multiple specifiers match the same app, it's only returned 1x in the returned slice.
func (a *Loader) LoadApps(specifier ...string) ([]*App, error) {
	names, cfgPaths, star := splitSpecifiers(specifier)

	if star || len(specifier) == 0 {
		return a.allApps()
	}

	result := make([]*App, 0, len(specifier))
	for _, path := range cfgPaths {
		app, err := a.AppPath(path)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}

		result = append(result, app)
	}

	apps, err := a.AppNames(names...)
	if err != nil {
		return nil, err
	}

	result = append(result, apps...)

	return dedupApps(result), nil
}

// AppNames discovers and loads the apps with the given names.
func (a *Loader) AppNames(names ...string) ([]*App, error) {
	namesMap := make(map[string]struct{}, len(names))
	result := make([]*App, 0, len(names))

	a.logger.Debugf("loader: loading app %q", names)

	for _, name := range names {
		namesMap[name] = struct{}{}
	}

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

		if _, exist := namesMap[appCfg.Name]; !exist {
			continue
		}

		app, err := a.fromCfg(appCfg)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}

		result = append(result, app)

		delete(namesMap, appCfg.Name)
	}

	notFoundApps := make([]string, 0, len(namesMap))
	for name := range namesMap {
		notFoundApps = append(notFoundApps, name)
	}

	if len(notFoundApps) != 0 {
		return nil, fmt.Errorf("could not find the following apps: %s", strings.Join(notFoundApps, ", "))
	}

	return result, nil
}

func (a *Loader) allApps() ([]*App, error) {
	a.logger.Debugf("loader: loading all apps")

	result := make([]*App, 0, len(a.appConfigPaths))

	for _, path := range a.appConfigPaths {
		app, err := a.AppPath(path)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}

		result = append(result, app)
	}

	return result, nil
}

// AppPath loads the app from the config file.
func (a *Loader) AppPath(appConfigPath string) (*App, error) {
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

func (a *Loader) taskSpec(apps []*App, spec string) []*Task {
	var result []*Task

	for _, app := range apps {
		if spec == "*" {
			result = append(result, app.Tasks()...)

			continue
		}

		for _, task := range app.Tasks() {
			if task.Name == spec {
				result = append(result, task)
			}
		}
	}

	return result
}

func (a *Loader) fromCfg(appCfg *cfg.App) (*App, error) {
	includeResolvers := IncludeCfgVarResolvers(a.repositoryRoot, appCfg.Name)

	err := appCfg.Merge(a.includeDB, includeResolvers)
	if err != nil {
		return nil, fmt.Errorf("merging includes failed: %w", err)
	}

	resolvers := DefaultAppCfgResolvers(a.repositoryRoot, appCfg.Name, a.gitCommitIDFunc)
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

// IsAppDirectory returns true and the path to the app config file if the
// directory contains an app config file.
func IsAppDirectory(dir string) (string, bool) {
	cfgPath := filepath.Join(dir, AppCfgFile)
	isFile, _ := fs.IsFile(cfgPath)

	return cfgPath, isFile
}

func findAppConfigs(searchDirs []string, searchDepth int) ([]string, error) {
	var result []string

	for _, searchDir := range searchDirs {
		if err := fs.DirsExist(searchDir); err != nil {
			return nil, fmt.Errorf("application search directory: %w", err)
		}

		cfgPaths, err := fs.FindFilesInSubDir(searchDir, AppCfgFile, searchDepth)
		if err != nil {
			return nil, err
		}

		result = append(result, cfgPaths...)
	}

	return result, nil
}

func dedupApps(apps []*App) []*App {
	dedupMap := make(map[string]*App, len(apps))

	for _, app := range apps {
		if _, exist := dedupMap[app.Path]; exist {
			continue
		}

		dedupMap[app.Path] = app
	}

	result := make([]*App, 0, len(dedupMap))

	for _, app := range dedupMap {
		result = append(result, app)
	}

	return result
}
