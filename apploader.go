package baur

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/simplesurance/baur/cfg"
	"github.com/simplesurance/baur/fs"
)

type Logger interface {
	Debugf(format string, v ...interface{})
}

// AppLoader discovers and instantiates apps in a repository.
type AppLoader struct {
	logger          Logger
	includeDB       *cfg.IncludeDB
	repositoryRoot  string
	appConfigPaths  []string
	gitCommitIDFunc func() (string, error)
}

// NewAppLoader instantiates an AppLoader.
// When an app config is loaded the DefaultResolvers are applied on the content
// before they are merged with their includes.
// The gitCommitIDFunc is used as config resolved to resolve $GITCOMMIT variables.
func NewAppLoader(repoCfg *cfg.Repository, gitCommitIDFunc func() (string, error), logger Logger) (*AppLoader, error) {
	repositoryRootDir := filepath.Dir(repoCfg.FilePath())

	appConfigPaths, err := findAppConfigs(fs.AbsPaths(repositoryRootDir, repoCfg.Discover.Dirs), repoCfg.Discover.SearchDepth)
	if err != nil {
		return nil, fmt.Errorf("discovering application config files failed: %w", err)
	}

	logger.Debugf("apploader: found the following application configs:\n%s", strings.Join(appConfigPaths, "\n"))

	return &AppLoader{
		logger:          logger,
		repositoryRoot:  repositoryRootDir,
		includeDB:       cfg.NewIncludeDB(logger),
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

// Load loads the apps that match the passed specifiers.
// Valid specifiers are:
// - application directory path
// - <APP-NAME>
// - '*'
// If multiple specifiers match the same app, it's only returned 1x in the returned slice.
func (a *AppLoader) Load(specifier ...string) ([]*App, error) {
	names, cfgPaths, star := splitSpecifiers(specifier)

	if star {
		return a.All()
	}

	result := make([]*App, 0, len(specifier))
	for _, path := range cfgPaths {
		app, err := a.Path(path)
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
func (a *AppLoader) AppNames(names ...string) ([]*App, error) {
	namesMap := make(map[string]struct{}, len(names))
	result := make([]*App, 0, len(names))

	a.logger.Debugf("apploader: loading app %q", names)

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
		return nil, fmt.Errorf("could not find the following apps: %s", strings.Join(names, ", "))
	}

	return result, nil
}

// All loads all apps in the repository.
func (a *AppLoader) All() ([]*App, error) {
	a.logger.Debugf("apploader: loading all apps")

	result := make([]*App, 0, len(a.appConfigPaths))

	for _, path := range a.appConfigPaths {
		app, err := a.Path(path)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}

		result = append(result, app)
	}

	return result, nil
}

// Path loads the app from the config file.
func (a *AppLoader) Path(appConfigPath string) (*App, error) {
	a.logger.Debugf("apploader: loading app from %q", appConfigPath)

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

func (a *AppLoader) fromCfg(appCfg *cfg.App) (*App, error) {
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
	cfgPath := path.Join(dir, AppCfgFile)
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
