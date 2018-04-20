package cfg

import (
	"io/ioutil"
	"os"

	toml "github.com/pelletier/go-toml"
	"github.com/pkg/errors"
)

//  AppFile contains the name of application configuration files
const AppFile = ".app.toml"

// App stores an application configuration.
type App struct {
	Name string `comment:"name of the application"`
}

// AppFromFile reads a application configuration file and returns it
func AppFromFile(path string) (*App, error) {
	config := App{}

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

// NewAppFile writes an exemplary Application configuration file to
// filepath. The name setting is set to appName
func (a *App) ToFile(filepath string) error {
	data, err := toml.Marshal(a)
	if err != nil {
		return errors.Wrapf(err, "marshalling failed")
	}

	f, err := os.OpenFile(filepath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		return err
	}

	_, err = f.Write(data)

	return err
}

// Validate validates a App configuration
func (a *App) Validate() error {
	if len(a.Name) == 0 {
		return errors.New("name parameter can not be empty")
	}

	return nil
}
