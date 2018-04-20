package app

import (
	"path"

	"github.com/pkg/errors"
	"github.com/simplesurance/baur/cfg"
)

type App struct {
	Name string
	Dir  string
}

func New(appPath string) (*App, error) {
	cfgPath := path.Join(appPath, cfg.AppFile)

	cfg, err := cfg.AppFromFile(cfgPath)
	if err != nil {
		return nil, errors.Wrapf(err, "reading %s failed", cfgPath)
	}

	return &App{
		Name: cfg.Name,
		Dir:  appPath,
	}, nil
}
