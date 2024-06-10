package baur

import (
	"fmt"
	"path/filepath"

	"github.com/simplesurance/baur/v4/internal/fs"
	"github.com/simplesurance/baur/v4/pkg/cfg"
)

// Repository represents a baur repository.
type Repository struct {
	Path        string
	CfgPath     string
	Cfg         *cfg.Repository
	SearchDepth int
}

// FindRepositoryCfg searches for a repository config file. The search starts
// in dir and traverses the parent directory down to the root.
// It returns the path to the first found repository configuration file.
func FindRepositoryCfg(dir string) (string, error) {
	return fs.FindFileInParentDirs(dir, RepositoryCfgFile)
}

// NewRepository parses the repository configuration file cfgPath and returns a
// Repository.
func NewRepository(cfgPath string) (*Repository, error) {
	realCfgPath, err := fs.RealPath(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("canonicalizing repository config path %q failed: %w", cfgPath, err)
	}

	repoCfg, err := cfg.RepositoryFromFile(realCfgPath)
	if err != nil {
		return nil, fmt.Errorf(
			"reading repository config %q failed: %w", realCfgPath, err)
	}

	err = repoCfg.Validate()
	if err != nil {
		return nil, fmt.Errorf(
			"validating repository config %q failed: %w", realCfgPath, err)
	}
	repoPath := filepath.Dir(realCfgPath)

	r := Repository{
		Cfg:         repoCfg,
		CfgPath:     realCfgPath,
		Path:        repoPath,
		SearchDepth: repoCfg.Discover.SearchDepth,
	}

	return &r, nil
}
