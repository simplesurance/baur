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
	cfg, err := cfg.AppFromFile(cfgPath, defaultBuildCmd)
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

	return &App{
		Dir:      path.Dir(cfgPath),
		Name:     cfg.Name,
		BuildCmd: cfg.Build.BuildCmd,
	}, nil
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
func (a *App) Build() (error, *BuildResult) {
	startTime := time.Now()

	out, exitCode, err := exec.Command(a.Dir, a.BuildCmd)
	if err != nil {
		return err, nil
	}

	return nil, &BuildResult{
		Duration: time.Since(startTime),
		Output:   out,
		ExitCode: exitCode,
		Success:  exitCode == 0,
	}
}

// SortAppsByName sorts the apps in the slice by Name
func SortAppsByName(apps []*App) {
	sort.Slice(apps, func(i int, j int) bool {
		return apps[i].Name < apps[j].Name
	})
}
