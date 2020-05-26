package baur

import (
	"os"
	"path"

	"github.com/pkg/errors"

	"github.com/simplesurance/baur/cfg"
	"github.com/simplesurance/baur/fs"
)

// Repository represents an repository containing applications
type Repository struct {
	Path          string
	CfgPath       string
	Cfg           *cfg.Repository
	AppSearchDirs []string
	SearchDepth   int
	PSQLURL       string
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
		return nil, errors.Wrapf(err,
			"reading repository config %s failed", cfgPath)
	}

	err = repoCfg.Validate()
	if err != nil {
		return nil, errors.Wrapf(err,
			"validating repository config %q failed", cfgPath)
	}
	repoPath := path.Dir(cfgPath)

	r := Repository{
		Cfg:           repoCfg,
		CfgPath:       cfgPath,
		Path:          repoPath,
		AppSearchDirs: fs.PathsJoin(repoPath, repoCfg.Discover.Dirs),
		SearchDepth:   repoCfg.Discover.SearchDepth,
		PSQLURL:       repoCfg.Database.PGSQLURL,
	}

	err = fs.DirsExist(r.AppSearchDirs...)
	if err != nil {
		return nil, errors.Wrapf(err, "validating repository config %q failed, "+
			"application_dirs parameter is invalid", cfgPath)
	}

	return &r, nil
}
