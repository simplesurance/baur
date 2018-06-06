package cfg

import (
	"fmt"
	"io/ioutil"
	"os"

	toml "github.com/pelletier/go-toml"
	"github.com/pkg/errors"
)

// App stores an application configuration.
type App struct {
	Name           string `toml:"name" comment:"name of the application"`
	S3Artifact     []*S3Artifact
	DockerArtifact []*DockerArtifact
	BuildCommand   string `toml:"build_command" commented:"true" comment:"command to build the application, overwrites the parameter in the repository config"`
}

// S3Artifact describes where a file artifact should be uploaded to
type S3Artifact struct {
	Path     string `toml:"path" comment:"path of the artifact" commented:"true"`
	Bucket   string `toml:"bucket" comment:"name of the S3 bucket where the file is stored" commented:"true"`
	DestFile string `toml:"dest_file" comment:"name of the uploaded file in the repository, valid variables: $APPNAME (name of the application), $UUID (generated UUID)" commented:"true"`
}

// DockerArtifact describes where a docker container is uploaded to
type DockerArtifact struct {
	IDFile     string `toml:"idfile" comment:"path to a text file that exist after the build and contains the docker image id (docker build --iidfile)" commented:"true"`
	Repository string `toml:"repository" comment:"name of the docker repository, e.g: simplesurance/pdfrender" commented:"true"`
	Tag        string `toml:"tag" comment:"tag that should be applied to the image, valid variables: $APPNAME, $UUID" commented:"true"`
}

// ExampleApp returns an exemplary app cfg struct with the name set to the given value
func ExampleApp(name string) *App {
	return &App{
		Name:         name,
		BuildCommand: "make docker_dist",

		S3Artifact: []*S3Artifact{
			&S3Artifact{
				Path:     fmt.Sprintf("dist/%s.tar.xz", name),
				Bucket:   "fho-baur-test",
				DestFile: "$APPNAME-$UUID.tar.xz",
			},
		},
		DockerArtifact: []*DockerArtifact{
			&DockerArtifact{
				IDFile:     fmt.Sprintf("dist/%s-container.id", name),
				Repository: "dockerhub",
				Tag:        "$APPNAME:$UUID",
			},
		},
	}
}

// AppFromFile reads a application configuration file and returns it.
// If the buildCmd is not set in the App configuration it's set to
// defaultBuildCommand
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

// ToFile writes an exemplary Application configuration file to
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

	for _, ar := range a.DockerArtifact {
		if err := ar.Validate(); err != nil {
			return errors.Wrap(err, "[[DockerArtifact]] section contains errors")
		}
	}

	for _, ar := range a.S3Artifact {
		if err := ar.Validate(); err != nil {
			return errors.Wrap(err, "[[S3Artifact]] section contains errors")
		}
	}

	return nil
}

// Validate validates a [[S3Artifact]] section
func (f *S3Artifact) Validate() error {
	if len(f.DestFile) == 0 {
		return errors.New("destfile parameter can not be unset or empty")
	}

	if len(f.Path) == 0 {
		return errors.New("path parameter can not be unset or empty")
	}

	if len(f.Bucket) == 0 {
		return errors.New("bucket parameter can not be unset or empty")
	}

	return nil
}

// Validate validates a [[DockerArtifact]] section
func (d *DockerArtifact) Validate() error {
	if len(d.IDFile) == 0 {
		return errors.New("idfile parameter can not be unset or empty")
	}

	if len(d.Repository) == 0 {
		return errors.New("repository parameter can not be unset or empty")
	}

	if len(d.Tag) == 0 {
		return errors.New("tag parameter can not be unset or empty")
	}

	return nil
}
