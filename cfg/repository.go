package cfg

import (
	"fmt"
	"io/ioutil"

	"github.com/pelletier/go-toml"
	"github.com/pkg/errors"
)

const (
	minSearchDepth = 0
	maxSearchDepth = 10
	// configVersion identifies the format of the configuration files,
	// whenever an incompatible change is made, this number has to be
	// increased
	configVersion int = 2
)

// Repository contains the repository configuration.
type Repository struct {
	ConfigVersion int      `toml:"config_version" comment:"Version of baur configuration format"`
	IncludeDirs   []string `toml:"include_dirs" commented:"true" comment:"Directories that contain include files for app.toml files"`
	Database      Database `toml:"Database"`
	Discover      Discover `comment:"Application discovery settings"`
}

// Database contains database configuration
type Database struct {
	PGSQLURL string `toml:"postgresql_url" comment:"Connection string to the PostgreSQL database, see https://www.postgresql.org/docs/current/static/libpq-connect.html#LIBPQ-CONNSTRING"`
}

// Discover stores the [Discover] section of the repository configuration.
type Discover struct {
	Dirs        []string `toml:"application_dirs" comment:"List of directories containing applications, example: ['go/code', 'shop/']"`
	SearchDepth int      `toml:"search_depth" comment:"Descend at most SearchDepth levels to find application configs"`
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

		IncludeDirs: []string{
			"baur_includes/",
		},

		Discover: Discover{
			Dirs:        []string{"."},
			SearchDepth: 1,
		},

		Database: Database{
			PGSQLURL: "postgres://postgres@localhost:5432/baur?sslmode=disable&connect_timeout=5",
		},
	}
}

// ToFile writes an Repository configuration file to filepath.
// If overwrite is true an existent file will be overwriten. If it's false the
// function returns an error if the file exist.
func (r *Repository) ToFile(filepath string, overwrite bool) error {
	return toFile(r, filepath, overwrite)
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
