package baur

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/simplesurance/baur/v2/pkg/cfg"
)

// App represents an Application.
type App struct {
	RelPath string
	Path    string
	Name    string

	repositoryRootPath string

	cfg *cfg.App
}

// NewApp instantiates an App object based on an app configuration.
func NewApp(appCfg *cfg.App, repositoryRootPath string) (*App, error) {
	appDir := filepath.Dir(appCfg.FilePath())

	appRelPath, err := filepath.Rel(repositoryRootPath, appDir)
	if err != nil {
		return nil, fmt.Errorf("%s: resolving repository relative application path failed: %w", appCfg.Name, err)
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

// String returns the name of the app.
func (a *App) String() string {
	return a.Name
}

// Tasks instantiates Task objects for each defined task in the app's
// configuration.
func (a *App) Tasks() []*Task {
	result := make([]*Task, 0, len(a.cfg.Tasks))

	for _, taskCfg := range a.cfg.Tasks {
		task := NewTask(taskCfg, a.Name, a.repositoryRootPath, a.Path)
		result = append(result, task)
	}

	return result
}

// SortAppsByName sorts the slice application names.
func SortAppsByName(apps []*App) {
	sort.Slice(apps, func(i int, j int) bool {
		return apps[i].Name < apps[j].Name
	})
}
