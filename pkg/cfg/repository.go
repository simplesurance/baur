package cfg

import (
	"fmt"
	"os"

	"github.com/pelletier/go-toml"
)

const (
	minSearchDepth = 0
	maxSearchDepth = 10
	// Version identifies the format of the configuration files that the
	// package can parse. Whenever an incompatible change is made, the
	// Version number is increased.
	Version int = 7
)

// Repository contains the repository configuration.
type Repository struct {
	ConfigVersion int `toml:"config_version" comment:"Internal field, version of baur configuration format"`

	Database Database
	Discover Discover

	filePath string
}

// Database contains database configuration
type Database struct {
	PGSQLURL string `toml:"postgresql_url" comment:"PostgreSQL database Connection string (https://www.postgresql.org/docs/current/static/libpq-connect.html#LIBPQ-CONNSTRING)\n The setting is overwritten by the environment variable BAUR_POSTGRESQL_URL."`
}

// Discover stores the [Discover] section of the repository configuration.
type Discover struct {
	Dirs        []string `toml:"application_dirs" comment:"Directories in which applications (.app.toml files) are discovered"`
	SearchDepth int      `toml:"search_depth" comment:"Descend at most search_depth levels to find application configs"`
}

// RepositoryFromFile reads the repository config from a file and returns it.
func RepositoryFromFile(cfgPath string) (*Repository, error) {
	config := Repository{}

	content, err := os.ReadFile(cfgPath)
	if err != nil {
		return nil, err
	}

	err = toml.Unmarshal(content, &config)
	if err != nil {
		return nil, err
	}

	config.filePath = cfgPath

	return &config, err
}

// ExampleRepository returns an exemplary Repository config
func ExampleRepository() *Repository {
	return &Repository{
		ConfigVersion: Version,

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
func (r *Repository) ToFile(filepath string, opts ...toFileOpt) error {
	return toFile(r, filepath, opts...)
}

func (r *Repository) FilePath() string {
	return r.filePath
}

// Validate validates a repository configuration
func (r *Repository) Validate() error {
	if r.ConfigVersion == 0 {
		return newFieldError("can not be unset or 0", "config_version")
	}
	if r.ConfigVersion != Version {
		return fmt.Errorf("incompatible configuration files\n"+
			"config_version value is %d, expecting version: %d\n"+
			"Update your baur configuration files or downgrade baur.", r.ConfigVersion, Version)
	}

	err := r.Discover.validate()
	if err != nil {
		return fieldErrorWrap(err, "Discover")
	}

	return nil
}

// validate validates the Discover section and sets defaults.
func (d *Discover) validate() error {
	if len(d.Dirs) == 0 {
		return newFieldError("can not be empty", "application_dirs")
	}

	if d.SearchDepth < minSearchDepth || d.SearchDepth > maxSearchDepth {
		return newFieldError(fmt.Sprintf("search_depth parameter must be in range (%d, %d]",
			minSearchDepth, maxSearchDepth),
			"search_depth",
		)
	}

	return nil
}
