package cfg

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	toml "github.com/pelletier/go-toml"
	"github.com/pkg/errors"
)

// App stores an application configuration.
type App struct {
	Name           string `toml:"name" comment:"name of the application"`
	S3Artifact     []*S3Artifact
	DockerArtifact []*DockerArtifact
	BuildCommand   string         `toml:"build_command" commented:"true" comment:"command to build the application, overwrites the parameter in the repository config"`
	SourceFiles    FileSources    `comment:"paths to file that affect the produces build artifacts, e.g: source files, the used compiler binary "`
	GitSourceFiles GitSourceFiles `comment:"If the baur repository is part of a git repository, this option can be used to specify source files tracked by git."`
}

// FileSources describes a file source
type FileSources struct {
	Paths []string `toml:"paths" commented:"true" comment:"relative path to source files,  supports Golang Glob syntax (https://golang.org/pkg/path/filepath/#Match) and ** to match files recursively"`
}

// GitSourceFiles describes source files that are in the git repository by git
// pathnames
type GitSourceFiles struct {
	// TODO: improve description
	Paths []string `toml:"paths" commented:"true" comment:"Specifies relative paths to source files that are tracked in the git repository.\n All paths must be inside the git repository.\n All patterns in pathnames are supported that git commands support.\n Files that are not tracked by the git repository are ignored. Tracked but modified files are matched."`
}

// S3Artifact describes where a file artifact should be uploaded to
type S3Artifact struct {
	Path     string `toml:"path" comment:"path of the artifact" commented:"true"`
	Bucket   string `toml:"bucket" comment:"name of the S3 bucket where the file is stored" commented:"true"`
	DestFile string `toml:"dest_file" comment:"name of the uploaded file in the repository, valid variables: $APPNAME, $UUID, $GITCOMMIT" commented:"true"`
}

// DockerArtifact describes where a docker container is uploaded to
type DockerArtifact struct {
	IDFile     string `toml:"idfile" comment:"path to a text file that exist after the build and contains the docker image id (docker build --iidfile)" commented:"true"`
	Repository string `toml:"repository" comment:"name of the docker repository" commented:"true"`
	Tag        string `toml:"tag" comment:"tag that should be applied to the image, valid variables: $APPNAME, $UUID, $GITCOMMIT"  commented:"true"`
}

// ExampleApp returns an exemplary app cfg struct with the name set to the given value
func ExampleApp(name string) *App {
	return &App{
		Name:         name,
		BuildCommand: "make docker_dist",
		SourceFiles: FileSources{
			Paths: []string{"Makefile", "src/**", "../../protos/src/pb/certificate.proto"},
		},
		GitSourceFiles: GitSourceFiles{
			Paths: []string{".", "../components/", "../../makeincludes/*.mk", "../../makeincludes/ui"},
		},
		S3Artifact: []*S3Artifact{
			&S3Artifact{
				Path:     fmt.Sprintf("dist/%s.tar.xz", name),
				Bucket:   "sisu-resources",
				DestFile: "$APPNAME-$GITCOMMIT.tar.xz",
			},
		},
		DockerArtifact: []*DockerArtifact{
			&DockerArtifact{
				IDFile:     fmt.Sprintf("%s-container.id", name),
				Repository: fmt.Sprintf("simplesurance/%s", name),
				Tag:        "$GITCOMMIT",
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

	removeEmptyRepositories(&config)

	return &config, err
}

// removeEmptyRepositories removes artifacts with only empty values.
// This is a woraround for https://github.com/pelletier/go-toml/issues/216
// It prevents that slices are commented in created Example configurations.
// Our validation function would fail for those and we would have to do
// exceptions for Artifacts with only empty values.
// Workaround: remove them from the config
func removeEmptyRepositories(a *App) {
	s3Arts := make([]*S3Artifact, 0, len(a.S3Artifact))
	dockerArts := make([]*DockerArtifact, 0, len(a.DockerArtifact))

	for _, s := range a.S3Artifact {
		if len(s.Bucket) == 0 &&
			len(s.DestFile) == 0 &&
			len(s.Path) == 0 {

			continue
		}

		s3Arts = append(s3Arts, s)
	}

	for _, d := range a.DockerArtifact {
		if len(d.IDFile) == 0 &&
			len(d.Repository) == 0 &&
			len(d.Tag) == 0 {

			continue
		}

		dockerArts = append(dockerArts, d)
	}

	a.S3Artifact = s3Arts
	a.DockerArtifact = dockerArts
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

	if err := a.SourceFiles.Validate(); err != nil {
		return errors.Wrap(err, "[SourceFiles] section contains errors")
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

// Validate validates a [[Sources.Files]] section
func (f *FileSources) Validate() error {
	for _, path := range f.Paths {
		if len(path) == 0 {
			return errors.New("path can not be empty")
		}
		if strings.Count(path, "**") > 1 {
			return errors.New("'**' can only appear one time in a path")
		}
	}

	return nil
}
