package baur

import (
	"fmt"
	"path/filepath"

	"github.com/simplesurance/baur/v2/internal/fs"
	"github.com/simplesurance/baur/v2/pkg/cfg"
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
	repoCfg, err := cfg.RepositoryFromFile(cfgPath)
	if err != nil {
		return nil, fmt.Errorf(
			"reading repository config %s failed: %w", cfgPath, err)
	}

	err = repoCfg.Validate()
	if err != nil {
		return nil, fmt.Errorf(
			"validating repository config %q failed: %w", cfgPath, err)
	}
	repoPath := filepath.Dir(cfgPath)

	r := Repository{
		Cfg:         repoCfg,
		CfgPath:     cfgPath,
		Path:        repoPath,
		SearchDepth: repoCfg.Discover.SearchDepth,
	}

	return &r, nil
}
