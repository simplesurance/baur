package baur

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/simplesurance/baur/v1/cfg"
	"github.com/simplesurance/baur/v1/internal/fs"
)

// Repository represents an repository containing applications
type Repository struct {
	Path        string
	CfgPath     string
	Cfg         *cfg.Repository
	SearchDepth int
	// TODO: remove PSQLURL
	PSQLURL string
}

// FindRepositoryCfg searches for a repository config file. The search starts in
// the passed directory and traverses the parent directory down to '/'.
// It returns the path to the first found repository configuration file.
func FindRepositoryCfg(dir string) (string, error) {
	return fs.FindFileInParentDirs(dir, RepositoryCfgFile)
}

// FindRepositoryCfgCwd searches for a repository config file in the current directory
// and all it's parents.
// It returns the path to the first found repository configuration file.
func FindRepositoryCfgCwd() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	return FindRepositoryCfg(cwd)
}

// NewRepository reads the configuration file and returns a Repository
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
		PSQLURL:     repoCfg.Database.PGSQLURL,
	}

	return &r, nil
}
