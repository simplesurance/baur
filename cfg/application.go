package cfg

import (
	"io/ioutil"
	"os"

	toml "github.com/pelletier/go-toml"
	"github.com/pkg/errors"
)

// App stores an application configuration.
type App struct {
	Name  string   `toml:"name",comment:"name of the application"`
	Build AppBuild `comment:"build configuration"`
}

// AppBuild contains application specific build settings
type AppBuild struct {
	BuildCmd string `toml:"build_command" commented:"true" comment:"command to build the application, if not set the BuildCommand from the repository config file is used. The command is run in the application diretory."`
}

// NewApp returns an exemplary app cfg struct with the name set to the given value
func ExampleApp(name string) *App {
	return &App{
		Name: name,
		Build: AppBuild{
			BuildCmd: "make",
		},
	}
}

// AppFromFile reads a application configuration file and returns it.
// If the buildCmd is not set in the App configuration it's set to
// defaultBuildCmd
func AppFromFile(path string, defaultBuildCmd string) (*App, error) {
	config := App{}

	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	err = toml.Unmarshal(content, &config)
	if err != nil {
		return nil, err
	}

	if len(config.Build.BuildCmd) == 0 {
		config.Build.BuildCmd = defaultBuildCmd
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

	err := a.Build.Validate()
	if err != nil {
		return errors.Wrap(err, "[Build] section contains errors")
	}

	return nil
}

// Validate validates the [Build] section of an application config file
func (b *AppBuild) Validate() error {
	if len(b.BuildCmd) == 0 {
		return errors.New("build_command parameter can not be empty")
	}

	return nil
}
