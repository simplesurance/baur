package cfg

import (
	"fmt"
	"io/ioutil"

	"github.com/pelletier/go-toml"
)

const (
	minSearchDepth = 0
	maxSearchDepth = 10
	// Version identifies the format of the configuration files,
	// whenever an incompatible change is made, this number has to be
	// increased
	Version int = 5
)

// Repository contains the repository configuration.
type Repository struct {
	ConfigVersion int `toml:"config_version" comment:"Version of baur configuration format"`

	Database Database
	Discover Discover `toml:"Discover" comment:"Application discovery settings"`

	filePath string
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

	config.filePath = cfgPath

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
			PGSQLURL: "postgres://postgres@localhost:5432/baur?sslmode=disable&connect_timeout=5",
		},
	}
}

// ToFile writes an Repository configuration file to filepath.
func (r *Repository) ToFile(filepath string, opts ...ToFileOpt) error {
	return toFile(r, filepath, opts...)
}

func (r *Repository) FilePath() string {
	return r.filePath
}

// Validate validates a repository configuration
func (r *Repository) Validate() error {
	if r.ConfigVersion == 0 {
		return NewFieldError("can not be unset or 0", "config_version")
	}
	if r.ConfigVersion != configVersion {
		return NewFieldError(
			fmt.Sprintf("incompatible configuration files\n"+
				"config_version value is %d, expecting version: %d\n"+
				"Update your baur configuration files or downgrade baur.", r.ConfigVersion, configVersion),
			"config_version",
		)
	}

	err := r.Discover.Validate()
	if err != nil {
		return FieldErrorWrap(err, "Discover")
	}

	return nil
}

// Validate validates the Discover section and sets defaults.
func (d *Discover) Validate() error {
	if len(d.Dirs) == 0 {
		return NewFieldError("can not be empty", "application_dirs")
	}

	if d.SearchDepth < minSearchDepth || d.SearchDepth > maxSearchDepth {
		return NewFieldError(fmt.Sprintf("search_depth parameter must be in range (%d, %d]",
			minSearchDepth, maxSearchDepth),
			"search_depth",
		)
	}

	return nil
}
