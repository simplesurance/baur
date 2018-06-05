package baur

import (
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/simplesurance/baur/cfg"
	"github.com/simplesurance/baur/fs"
	"github.com/simplesurance/baur/log"
	"github.com/simplesurance/baur/version"
)

// Repository represents an repository containing applications
type Repository struct {
	Path            string
	CfgPath         string
	AppSearchDirs   []string
	SearchDepth     int
	DefaultBuildCmd string
}

// FindRepository searches for a repository config file in the current directory
// and all it's parents. If a repository config file is found it returns a
// Repository
func FindRepository() (*Repository, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	rootPath, err := fs.FindFileInParentDirs(cwd, RepositoryCfgFile)
	if err != nil {
		return nil, err
	}

	return NewRepository(rootPath)
}

func ensureRepositoryCFGHasVersion(cfg *cfg.Repository, cfgPath string) error {
	if cfg.BaurVersion == "" {
		cfg.BaurVersion = version.Version

		err := cfg.ToFile(cfgPath, true)
		if err != nil {
			return err
		}

		log.Debugf("written baur version to repository config %s\n", cfgPath)
	}

	return nil
}

// NewRepository reads the configuration file and returns a Repository
func NewRepository(cfgPath string) (*Repository, error) {
	cfg, err := cfg.RepositoryFromFile(cfgPath)
	if err != nil {
		return nil, errors.Wrapf(err,
			"reading repository config %s failed", cfgPath)
	}

	err = ensureRepositoryCFGHasVersion(cfg, cfgPath)
	if err != nil {
		return nil, errors.Wrapf(err, "updating baur_version in %s failed", cfgPath)
	}

	err = cfg.Validate()
	if err != nil {
		return nil, errors.Wrapf(err,
			"validating repository config %s failed",
			cfgPath)
	}

	return &Repository{
		CfgPath:         cfgPath,
		DefaultBuildCmd: cfg.Build.BuildCmd,
		Path:            path.Dir(cfgPath),
		AppSearchDirs:   fs.PathsJoin(path.Dir(cfgPath), cfg.Discover.Dirs),
		SearchDepth:     cfg.Discover.SearchDepth,
	}, nil
}

// FindApps searches for application config files in the AppSearchDirs of the
// repository and returns all found apps
func (r *Repository) FindApps() ([]*App, error) {
	var result []*App

	for _, searchDir := range r.AppSearchDirs {
		appsCfgPaths, err := fs.FindFilesInSubDir(searchDir, AppCfgFile, r.SearchDepth)
		if err != nil {
			return nil, errors.Wrap(err, "finding application configs failed")
		}

		for _, appCfgPath := range appsCfgPaths {
			a, err := NewApp(appCfgPath, r.DefaultBuildCmd)
			if err != nil {
				return nil, err
			}

			result = append(result, a)
		}
	}

	return result, nil
}

// AppByDir reads an application config file from the direcory and returns an
// App
func (r *Repository) AppByDir(appDir string) (*App, error) {
	cfgPath := path.Join(appDir, AppCfgFile)

	cfgPath, err := filepath.Abs(cfgPath)
	if err != nil {
		return nil, err
	}

	return NewApp(cfgPath, r.DefaultBuildCmd)
}

// AppByName searches for an App with the given name in the repository and
// returns it. If none is found os.ErrNotExist is returned.
func (r *Repository) AppByName(name string) (*App, error) {
	for _, searchDir := range r.AppSearchDirs {
		appsCfgPaths, err := fs.FindFilesInSubDir(searchDir, AppCfgFile, r.SearchDepth)
		if err != nil {
			return nil, errors.Wrap(err, "finding application failed")
		}

		for _, appCfgPath := range appsCfgPaths {
			a, err := NewApp(appCfgPath, r.DefaultBuildCmd)
			if err != nil {
				return nil, err
			}
			if a.Name == name {
				return a, nil
			}
		}
	}

	return nil, os.ErrNotExist
}
