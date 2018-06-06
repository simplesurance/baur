package baur

import (
	"path"
	"sort"
	"time"

	"github.com/pkg/errors"
	"github.com/simplesurance/baur/cfg"
	"github.com/simplesurance/baur/exec"
)

// App represents an application
type App struct {
	Dir      string
	Name     string
	BuildCmd string
}

// NewApp reads the configuration file and returns a new App
func NewApp(cfgPath, defaultBuildCmd string) (*App, error) {
	cfg, err := cfg.AppFromFile(cfgPath)
	if err != nil {
		return nil, errors.Wrapf(err,
			"reading application config %s failed", cfgPath)
	}

	err = cfg.Validate()
	if err != nil {
		return nil, errors.Wrapf(err,
			"validating application config %s failed",
			cfgPath)
	}

	app := App{
		Dir:      path.Dir(cfgPath),
		Name:     cfg.Name,
		BuildCmd: cfg.Build.Command,
	}

	if len(app.BuildCmd) == 0 {
		app.BuildCmd = defaultBuildCmd
	}

	return &app, nil
}

// BuildResult contains the result of build
type BuildResult struct {
	Duration time.Duration
	ExitCode int
	Output   string
	Success  bool
}

// Build builds an application by executing it's BuildCmd in the application
// directory
func (a *App) Build() (*BuildResult, error) {
	startTime := time.Now()

	out, exitCode, err := exec.Command(a.Dir, a.BuildCmd)
	if err != nil {
		return nil, err
	}

	return &BuildResult{
		Duration: time.Since(startTime),
		Output:   out,
		ExitCode: exitCode,
		Success:  exitCode == 0,
	}, nil
}

// SortAppsByName sorts the apps in the slice by Name
func SortAppsByName(apps []*App) {
	sort.Slice(apps, func(i int, j int) bool {
		return apps[i].Name < apps[j].Name
	})
}
