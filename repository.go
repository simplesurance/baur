package baur

import (
	"os"
	"path"

	"github.com/pkg/errors"

	"github.com/simplesurance/baur/cfg"
	"github.com/simplesurance/baur/fs"
	"github.com/simplesurance/baur/log"
)

// Repository represents an repository containing applications
type Repository struct {
	Path          string
	CfgPath       string
	Cfg           *cfg.Repository
	AppSearchDirs []string
	SearchDepth   int
	PSQLURL       string
	includeDB     *cfg.IncludeDB
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
		includeDB:     cfg.NewIncludeDB(log.StdLogger),
	}

	err = fs.DirsExist(r.AppSearchDirs...)
	if err != nil {
		return nil, errors.Wrapf(err, "validating repository config %q failed, "+
			"application_dirs parameter is invalid", cfgPath)
	}

	return &r, nil
}
