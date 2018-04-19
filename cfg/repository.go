package cfg

import (
	"fmt"
	"io/ioutil"
	"os"

	toml "github.com/pelletier/go-toml"
	"github.com/pkg/errors"
	"github.com/simplesurance/baur/fs"
)

// RepositoryFile contains the name of the repository configuration file.
const RepositoryFile = ".baur.toml"

const (
	MinSearchDepth = 1
	MaxSearchDepth = 10
)

// Repository contains the repository configuration.
type Repository struct {
	Discover *Discover `comment:"application discovery settings"`
}

// Discover stores the [Discover] section of the repository configuration.
type Discover struct {
	Dirs        []string `toml:"application_dirs" comment:"list of directories containing applications, example: ['go/code', 'shop/']"`
	SearchDepth int      `toml:"search_depth" comment:"specifies the max. directory the application search recurses into subdirectories"`
}

// NewRepositoryFile writes an exemplary Repository configuration file to
// filepath
func NewRepositoryFile(filepath string) error {
	data, err := toml.Marshal(
		Repository{Discover: &Discover{
			SearchDepth: 1,
		},
		})
	if err != nil {
		return errors.Wrapf(err, "marshalling Repository config failed")
	}

	f, err := os.OpenFile(filepath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		return err
	}

	_, err = f.Write(data)

	return err
}

// RepositoryFromFile reads the repository config from a file and returns it.
func RepositoryFromFile(path string) (*Repository, error) {
	config := Repository{}

	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	err = toml.Unmarshal(content, &config)
	if err != nil {
		return nil, err
	}

	return &config, err
}

// Validate validates a repository configuration
func (r *Repository) Validate() error {
	err := r.Discover.Validate()
	if err != nil {
		return errors.Wrapf(err, "[Discover] section contains errors")
	}

	return nil
}

// Validate validates the Discover section and sets defaults.
func (d *Discover) Validate() error {
	if len(d.Dirs) == 0 {
		return fmt.Errorf("application_dirs parameter " +
			"in [Discover] sesction is empty")
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
