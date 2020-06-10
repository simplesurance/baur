package baur

import (
	"path"
	"path/filepath"
	"sort"

	"github.com/pkg/errors"

	"github.com/simplesurance/baur/cfg"
)

// App represents an application
type App struct {
	RelPath string
	Path    string
	Name    string

	repositoryRootPath string

	cfg *cfg.App
}

// NewApp reads the configuration file and returns a new App
func NewApp(appCfg *cfg.App, repositoryRootPath string) (*App, error) {
	appDir := path.Dir(appCfg.FilePath())

	appRelPath, err := filepath.Rel(repositoryRootPath, appDir)
	if err != nil {
		return nil, errors.Wrapf(err, "%s: resolving repository relative application path failed", appCfg.Name)
	}

	app := App{
		cfg:                appCfg,
		Path:               appDir,
		RelPath:            appRelPath,
		Name:               appCfg.Name,
		repositoryRootPath: repositoryRootPath,
	}

	return &app, nil
}

// String returns the string representation of an app
func (a *App) String() string {
	return a.Name
}

func (a *App) Tasks() []*Task {
	result := make([]*Task, 0, len(a.cfg.Tasks))

	for _, taskCfg := range a.cfg.Tasks {
		task := NewTask(taskCfg, a.Name, a.repositoryRootPath, a.Path)
		result = append(result, task)
	}

	return result
}

// SortAppsByName sorts the apps in the slice by Name
func SortAppsByName(apps []*App) {
	sort.Slice(apps, func(i int, j int) bool {
		return apps[i].Name < apps[j].Name
	})
}
