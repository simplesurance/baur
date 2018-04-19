package cfg

import (
	"errors"
	"io/ioutil"

	toml "github.com/pelletier/go-toml"
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
