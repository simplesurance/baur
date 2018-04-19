package cfg

import (
	"io/ioutil"
	"os"

	toml "github.com/pelletier/go-toml"
	"github.com/pkg/errors"
)

// ApplicationFile contains the name of application configuration files
const ApplicationFile = ".app.toml"

// Application stores an application configuration.
type Application struct {
	Name string `comment:"name of the application"`
}

// ApplicationFromFile reads a application configuration file and returns it.
func ApplicationFromFile(path string) (*Application, error) {
	config := Application{}

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

// Validate validates a Application configuration.
func (a *Application) Validate() error {
	if len(a.Name) == 0 {
		return errors.New("name parameter can not be empty")
	}

	return nil
}

// NewApplicationFile writes an exemplary Application configuration file to
// filepath. The name setting is set to appName
func NewApplicationFile(appName, filepath string) error {
	data, err := toml.Marshal(Application{Name: appName})
	if err != nil {
		return errors.Wrapf(err, "marshalling Application failed")
	}

	f, err := os.OpenFile(filepath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		return err
	}

	_, err = f.Write(data)

	return err
}
