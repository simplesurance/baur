package cfg

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	toml "github.com/pelletier/go-toml"
	"github.com/pkg/errors"
	"github.com/simplesurance/baur/fs"
)

const (
	MinSearchDepth = 1
	MaxSearchDepth = 10
)

// Repository contains the repository configuration.
type Repository struct {
	Discover Discover        `comment:"application discovery settings"`
	Build    RepositoryBuild `comment:"build configuration"`
}

// Discover stores the [Discover] section of the repository configuration.
type Discover struct {
	Dirs        []string `toml:"application_dirs" comment:"list of directories containing applications, example: ['go/code', 'shop/']"`
	SearchDepth int      `toml:"search_depth" comment:"specifies the max. directory the application search recurses into subdirectories"`
}

// RepositoryBuild contains the build section of the repository
type RepositoryBuild struct {
	BuildCmd string `toml:"build_command" comment:"command to build the application, can be overwritten in the application specific config files"`
}

// RepositoryFromFile reads the repository config from a file and returns it.
func RepositoryFromFile(cfgPath string) (*Repository, error) {
	config := Repository{}

	content, err := ioutil.ReadFile(cfgPath)
	if err != nil {
		return nil, err
	}

	err = toml.Unmarshal(content, &config)
	if err != nil {
		return nil, err
	}

	rootPath := path.Dir(cfgPath)
	config.Discover.absDirs(rootPath)

	return &config, err
}

// absDirs converts the path in Dirs to absolute Paths
func (d *Discover) absDirs(rootPath string) {
	absDiscoveryDirs := make([]string, 0, len(d.Dirs))

	for _, d := range d.Dirs {
		absDir := path.Clean(path.Join(rootPath, d))
		absDiscoveryDirs = append(absDiscoveryDirs, absDir)
	}

	d.Dirs = absDiscoveryDirs
}

// ExampleRepository returns an exemplary Repository config
func ExampleRepository() *Repository {
	return &Repository{
		Discover: Discover{
			SearchDepth: 1,
		},
		Build: RepositoryBuild{
			BuildCmd: "make docker_dist",
		},
	}
}

// ToFile writes an Repository configuration file to filepath
func (r *Repository) ToFile(filepath string) error {
	data, err := toml.Marshal(*r)
	if err != nil {
		return errors.Wrapf(err, "marshalling config failed")
	}

	f, err := os.OpenFile(filepath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		return err
	}

	_, err = f.Write(data)

	return err
}

// Validate validates a repository configuration
func (r *Repository) Validate() error {
	err := r.Discover.Validate()
	if err != nil {
		return errors.Wrapf(err, "[Discover] section contains errors")
	}

	err = r.Build.Validate()
	if err != nil {
		return errors.Wrapf(err, "[Build] section contains errors")
	}

	return nil
}

// Validate validates the Discover section and sets defaults.
func (d *Discover) Validate() error {
	if len(d.Dirs) == 0 {
		return fmt.Errorf("application_dirs parameter " +
			"is empty")
	}

	err := fs.DirsExist(d.Dirs)
	if err != nil {
		return errors.Wrap(err, "application_dirs parameter is invalid")
	}

	if d.SearchDepth < MinSearchDepth || d.SearchDepth > MaxSearchDepth {
		return fmt.Errorf("search_depth parameter must be in range (%d, %d]",
			MinSearchDepth, MaxSearchDepth)
	}

	return nil
}

// Validate validates the [Build] section of a repository config file
func (b *RepositoryBuild) Validate() error {
	if len(b.BuildCmd) == 0 {
		return errors.New("build_command can not be empty")
	}

	return nil
}
