package baur

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/simplesurance/baur/cfg"
	"github.com/simplesurance/baur/fs"
	"github.com/simplesurance/baur/git"
)

// Repository represents an repository containing applications
type Repository struct {
	Path               string
	CfgPath            string
	AppSearchDirs      []string
	SearchDepth        int
	gitCommitID        string
	gitWorktreeIsDirty *bool
	PSQLURL            string
	Includes           map[string]cfg.BuildInputInclude
}

// FindRepository searches for a repository config file. The search starts in
// the passed directory and traverses the parent directory down to '/'. The first found repository
// configuration file is returned.
func FindRepository(dir string) (*Repository, error) {
	rootPath, err := fs.FindFileInParentDirs(dir, RepositoryCfgFile)
	if err != nil {
		return nil, err
	}

	return NewRepository(rootPath)
}

// FindRepositoryCwd searches for a repository config file in the current directory
// and all it's parents. If a repository config file is found it returns a
// Repository
func FindRepositoryCwd() (*Repository, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return FindRepository(cwd)
}

func (r *Repository) populateIncludes(repoCfg *cfg.Repository) error {
	var includeFilePaths []string

	if len(repoCfg.IncludeDirs) == 0 {
		return nil
	}

	for _, incDir := range repoCfg.IncludeDirs {
		absIncDir := path.Join(r.Path, incDir)

		err := fs.DirsExist(absIncDir)
		if err != nil {
			return fmt.Errorf("include_dir %s does not exist in repository", incDir)
		}

		incFiles, err := fs.FindFilesInSubDir(absIncDir, "*.toml", 0)
		if err != nil {
			return errors.Wrap(err, "finding include files failed")
		}

		includeFilePaths = append(includeFilePaths, incFiles...)
	}

	r.Includes = make(map[string]cfg.BuildInputInclude, len(includeFilePaths))
	for _, incFile := range includeFilePaths {
		includeCfg, err := cfg.IncludeFromFile(incFile)
		if err != nil {
			return fmt.Errorf("reading include file %s failed", incFile)
		}

		for _, include := range includeCfg.BuildInput {
			if _, exist := r.Includes[include.ID]; exist {
				return fmt.Errorf("include id '%s' is used multiple times, include ids must be unique", include.ID)
			}
			r.Includes[include.ID] = include
		}
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

	err = cfg.Validate()
	if err != nil {
		return nil, errors.Wrapf(err,
			"validating repository config %q failed", cfgPath)
	}

	r := Repository{
		CfgPath:       cfgPath,
		Path:          path.Dir(cfgPath),
		AppSearchDirs: fs.PathsJoin(path.Dir(cfgPath), cfg.Discover.Dirs),
		SearchDepth:   cfg.Discover.SearchDepth,
		PSQLURL:       cfg.Database.PGSQLURL,
	}

	err = fs.DirsExist(r.AppSearchDirs...)
	if err != nil {
		return nil, errors.Wrapf(err, "validating repository config %q failed, "+
			"application_dirs parameter is invalid", cfgPath)
	}

	err = r.populateIncludes(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "loading include files failed")
	}

	return &r, nil
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
			a, err := NewApp(r, appCfgPath)
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

	return NewApp(r, cfgPath)
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
			a, err := NewApp(r, appCfgPath)
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

// GitCommitID returns the Git commit ID in the baur repository root
func (r *Repository) GitCommitID() (string, error) {
	if len(r.gitCommitID) != 0 {
		return r.gitCommitID, nil
	}

	commit, err := git.CommitID(r.Path)
	if err != nil {
		return "", errors.Wrap(err, "determining Git commit ID failed, "+
			"ensure that the git command is in a directory in $PATH and "+
			"that the .baur.toml file is part of a git repository")
	}

	r.gitCommitID = commit

	return commit, nil
}

// GitWorkTreeIsDirty returns true if the git repository contains untracked
// changes
func (r *Repository) GitWorkTreeIsDirty() (bool, error) {
	if r.gitWorktreeIsDirty != nil {
		return *r.gitWorktreeIsDirty, nil
	}

	isDirty, err := git.WorkTreeIsDirty(r.Path)
	if err != nil {
		return false, errors.Wrap(err, "determining Git worktree state failed, "+
			"ensure that the git command is in a directory in $PATH and "+
			"that the .baur.toml file is part of a git repository")
	}

	r.gitWorktreeIsDirty = &isDirty

	return isDirty, nil
}
