package cfg

import (
	"io/ioutil"
	"os"

	toml "github.com/pelletier/go-toml"
	"github.com/pkg/errors"
	"github.com/simplesurance/baur"
)

// App stores an application configuration.
type App struct {
	Name string `comment:"name of the application"`
}

// AppFileReader implements the discover.AppCfgReader interface
type AppFileReader struct{}

// NewApp returns a new app cfg struct with the name set to the given value
func NewApp(name string) *App {
	return &App{Name: name}
}

// AppFromFile reads a application configuration file and returns it
func (a *AppFileReader) AppFromFile(path string) (baur.AppCfg, error) {
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
	data, err := toml.Marshal(*a)
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

// Name returns the name parameter of the app cfg
func (a *App) GetName() string {
	return a.Name
}
