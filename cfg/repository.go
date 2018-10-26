package cfg

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/pelletier/go-toml"
	"github.com/pkg/errors"
)

const (
	minSearchDepth = 0
	maxSearchDepth = 10
	// configVersion identifies the format of the configuration files,
	// whenever an incompatible change is made, this number has to be
	// increased
	configVersion int = 1
)

// Repository contains the repository configuration.
type Repository struct {
	Discover      Discover `comment:"application discovery settings"`
	ConfigVersion int      `toml:"config_version" comment:"internal, version of baur cfg file"`
	Database      Database `toml:"Database" comment:"configures the database in which build informations are stored"`
}

// Database contains database configuration
type Database struct {
	PGSQLURL string `toml:"postgresql_url" comment:"connection string to the PostgreSQL database, see https://www.postgresql.org/docs/current/static/libpq-connect.html#LIBPQ-CONNSTRING"`
}

// Discover stores the [Discover] section of the repository configuration.
type Discover struct {
	Dirs        []string `toml:"application_dirs" comment:"list of directories containing applications, example: ['go/code', 'shop/']"`
	SearchDepth int      `toml:"search_depth" comment:"specifies the max. directory the application search recurses into subdirectories"`
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

	return &config, err
}

// ExampleRepository returns an exemplary Repository config
func ExampleRepository() *Repository {
	return &Repository{
		ConfigVersion: configVersion,
		Discover: Discover{
			Dirs:        []string{"."},
			SearchDepth: 1,
		},

		Database: Database{
			PGSQLURL: "postgres://postgres@localhost:5432/baur?sslmode=disable",
		},
	}
}

// ToFile writes an Repository configuration file to filepath.
// If overwrite is true an existent file will be overwriten. If it's false the
// function returns an error if the file exist.
func (r *Repository) ToFile(filepath string, overwrite bool) error {
	var openFlags int

	data, err := toml.Marshal(*r)
	if err != nil {
		return errors.Wrap(err, "marshalling config failed")
	}

	if overwrite {
		openFlags = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	} else {
		openFlags = os.O_WRONLY | os.O_CREATE | os.O_EXCL
	}

	f, err := os.OpenFile(filepath, openFlags, 0666)
	if err != nil {
		return err
	}

	_, err = f.Write(data)
	if err != nil {
		return errors.Wrap(err, "writing to file failed")
	}

	err = f.Close()
	if err != nil {
		return errors.Wrap(err, "closing file failed")
	}

	return err
}

// Validate validates a repository configuration
func (r *Repository) Validate() error {
	if r.ConfigVersion == 0 {
		return fmt.Errorf("config_version value is unset or 0")
	}
	if r.ConfigVersion != configVersion {
		return fmt.Errorf("incompatible configuration files\n"+
			"config_version value is %d, expecting version: %d\n"+
			"Update your baur configuration files or downgrade baur.", r.ConfigVersion, configVersion)
	}

	err := r.Discover.Validate()
	if err != nil {
		return errors.Wrap(err, "[Discover] section contains errors")
	}

	return nil
}

// Validate validates the Discover section and sets defaults.
func (d *Discover) Validate() error {
	if len(d.Dirs) == 0 {
		return errors.New("application_dirs parameter is empty")
	}

	if d.SearchDepth < minSearchDepth || d.SearchDepth > maxSearchDepth {
		return fmt.Errorf("search_depth parameter must be in range (%d, %d]",
			minSearchDepth, maxSearchDepth)
	}

	return nil
}
