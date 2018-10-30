package cfg

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/pelletier/go-toml"
	"github.com/pkg/errors"
)

// App stores an application configuration.
type App struct {
	Name  string `toml:"name" comment:"name of the application"`
	Build Build  `toml:"Build"`
}

// Build the build section
type Build struct {
	Command string      `toml:"command" commented:"false" comment:"command to build the application"`
	Input   BuildInput  `comment:"specifies the inputs for an build, an input is everything that affects the build output"`
	Output  BuildOutput `comment:"specifies the outputs that the application build produces"`
}

// BuildInput contains information about build inputs
type BuildInput struct {
	Files         FileInputs    `comment:"file paths, e.g: source files, the used compiler binary"`
	GitFiles      GitFileInputs `comment:"If the baur repository is part of a git repository, this option can be used to specify source files tracked by git."`
	GolangSources GolangSources `comment:"Directories containing Golang applications, all source files to build the application are located and added as inputs (excluding stdlib and test files)"`
}

// GolangSources specifies inputs for Golang Applications
type GolangSources struct {
	Paths  []string `toml:"paths" comment:"paths to directories containing Golang source files" commented:"true"`
	GoPath string   `toml:"go_path" comment:"specifies the GOPATH, that is used for source file discovery, if not set or empty the current GOPATH is used. The go_path is relative to the application directory." commented:"true"`
}

// FileInputs describes a file source
type FileInputs struct {
	Paths []string `toml:"paths" commented:"false" comment:"relative path to source files,  supports Golang Glob syntax (https://golang.org/pkg/path/filepath/#Match) and ** to match files recursively"`
}

// GitFileInputs describes source files that are in the git repository by git
// pathnames
type GitFileInputs struct {
	// TODO: improve description
	Paths []string `toml:"paths" commented:"false" comment:"Specifies relative paths to source files that are tracked in the git repository.\n All paths must be inside the git repository.\n All patterns in pathnames are supported that git commands support.\n Files that are not tracked by the git repository are ignored. Tracked but modified files are matched."`
}

// BuildOutput the build output section
type BuildOutput struct {
	File        []*FileOutput        `comment:"a file that is produces by the build"`
	DockerImage []*DockerImageOutput `comment:"a docker image that is produced by the build"`
}

// FileOutput describes where a file artifact should be uploaded to
type FileOutput struct {
	Path     string   `toml:"path" comment:"path of the artifact" commented:"true"`
	S3Upload S3Upload `comment:"defines an S3 location that the file is uploaded to"`
}

// DockerImageRegistryUpload holds information about where the docker image
// should be uploaded to
type DockerImageRegistryUpload struct {
	Repository string `toml:"repository" comment:"docker repository path, format: [<server[:port]>/]<owner>/<repository>:<tag>" commented:"true"`
	Tag        string `toml:"tag" comment:"tag that is applied to the image, valid variables: $APPNAME, $UUID, $GITCOMMIT"  commented:"true"`
}

// S3Upload contains S3 upload information
type S3Upload struct {
	Bucket   string `toml:"bucket" comment:"S3 bucket name" commented:"true"`
	DestFile string `toml:"dest_file" comment:"name of the uploaded file in the repository, valid variables: $APPNAME, $UUID, $GITCOMMIT" commented:"true"`
}

// DockerImageOutput describes where a docker container is uploaded to
type DockerImageOutput struct {
	IDFile         string                    `toml:"idfile" comment:"path to a text file that exist after the build and contains the docker image id (docker build --iidfile)" commented:"true"`
	RegistryUpload DockerImageRegistryUpload `comment:"specifies where the image is uploaded to"`
}

// ExampleApp returns an exemplary app cfg struct with the name set to the given value
func ExampleApp(name string) *App {
	return &App{
		Name: name,

		Build: Build{
			Command: "make dist",
			Input: BuildInput{
				Files: FileInputs{
					Paths: []string{".app.toml"},
				},
				GitFiles: GitFileInputs{
					Paths: []string{"."},
				},
				GolangSources: GolangSources{
					Paths:  []string{"."},
					GoPath: "../",
				},
			},
			Output: BuildOutput{
				File: []*FileOutput{
					&FileOutput{
						Path: fmt.Sprintf("dist/%s.tar.xz", name),
						S3Upload: S3Upload{
							Bucket:   "sisu-resources/binaries/sisu",
							DestFile: "$APPNAME-$GITCOMMIT.tar.xz",
						},
					},
				},
				DockerImage: []*DockerImageOutput{
					&DockerImageOutput{
						IDFile: fmt.Sprintf("%s-container.id", name),
						RegistryUpload: DockerImageRegistryUpload{
							Repository: fmt.Sprintf("simplesurance/%s", name),
							Tag:        "$GITCOMMIT",
						},
					},
				},
			},
		},
	}
}

// AppFromFile reads a application configuration file and returns it.
// If the buildCmd is not set in the App configuration it's set to
// defaultBuild.Command
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

	removeEmptySections(&config)

	return &config, err
}

// removeEmptySections removes elements from slices of the that are empty.
// This is a workaround for https://github.com/pelletier/go-toml/issues/216
// It prevents that slices are commented in created Example configurations.
// To prevent that we have empty elements in the slice that we process later and
// validate, remove them from the config
func removeEmptySections(a *App) {
	fileOutputs := make([]*FileOutput, 0, len(a.Build.Output.File))
	dockerImageOutputs := make([]*DockerImageOutput, 0, len(a.Build.Output.DockerImage))

	for _, f := range a.Build.Output.File {
		if f.IsEmpty() {
			continue
		}

		fileOutputs = append(fileOutputs, f)
	}

	for _, d := range a.Build.Output.DockerImage {
		if d.IsEmpty() {
			continue
		}

		dockerImageOutputs = append(dockerImageOutputs, d)
	}

	a.Build.Output.File = fileOutputs
	a.Build.Output.DockerImage = dockerImageOutputs
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
	if err != nil {
		return errors.Wrap(err, "writing to file failed")
	}

	err = f.Close()
	if err != nil {
		return errors.Wrap(err, "closing file failed")
	}

	return err
}

// Validate validates a App configuration
func (a *App) Validate() error {
	if len(a.Name) == 0 {
		return errors.New("name parameter can not be empty")
	}

	if err := a.Build.Validate(); err != nil {
		return err
	}

	return nil

}

// Validate validates the build section
func (b *Build) Validate() error {
	if len(b.Command) == 0 {
		return errors.New("[Build] section contains errors: command can not be empty")
	}
	if err := b.Input.Validate(); err != nil {
		return err
	}

	if err := b.Output.Validate(); err != nil {
		return err
	}

	return nil
}

// Validate validates the BuildInput section
func (b *BuildInput) Validate() error {
	if err := b.Files.Validate(); err != nil {
		return errors.Wrap(err, "[Build.Input.Files] section contains errors")
	}

	if err := b.GolangSources.Validate(); err != nil {
		return errors.Wrap(err, "[Build.Input.Files] section contains errors")
	}

	// TODO: add validation for gitfiles section

	return nil
}

// Validate validates the GolangSources section
func (g *GolangSources) Validate() error {
	for _, p := range g.Paths {
		if len(p) == 0 {
			return errors.New("a path can not be empty")
		}
	}

	return nil
}

// Validate validates the BuildOutput section
func (b *BuildOutput) Validate() error {
	for _, f := range b.File {
		if err := f.Validate(); err != nil {
			return errors.Wrap(err, "[[Build.Output.File]] section contains errors")
		}
	}

	for _, d := range b.DockerImage {
		if err := d.Validate(); err != nil {
			return errors.Wrap(err, "[[Build.Output.DockerImage]] section contains errors")
		}
	}

	return nil
}

// IsEmpty returns true if FileOutput is empty
func (f *FileOutput) IsEmpty() bool {
	return len(f.Path) == 0 && f.S3Upload.IsEmpty()
}

// IsEmpty returns true if S3Upload is empty
func (s *S3Upload) IsEmpty() bool {
	return len(s.Bucket) == 0 && len(s.DestFile) == 0
}

// Validate validates a [[Build.Output.File]] section
func (f *FileOutput) Validate() error {
	if len(f.Path) == 0 {
		return errors.New("path parameter can not be unset or empty")
	}

	return f.S3Upload.Validate()
}

//IsEmpty returns true if the struct is empty
func (d *DockerImageRegistryUpload) IsEmpty() bool {
	return len(d.Repository) == 0 && len(d.Tag) == 0
}

// IsEmpty returns true if DockerImageOutput is empty
func (d *DockerImageOutput) IsEmpty() bool {
	return len(d.IDFile) == 0 && d.RegistryUpload.IsEmpty()

}

// Validate validates a [[Build.Output.File]] section
func (s *S3Upload) Validate() error {
	if len(s.DestFile) == 0 {
		return errors.New("destfile parameter can not be unset or empty")
	}

	if len(s.Bucket) == 0 {
		return errors.New("bucket parameter can not be unset or empty")
	}

	return nil
}

// Validate validates its content
func (d *DockerImageOutput) Validate() error {
	if len(d.IDFile) == 0 {
		return errors.New("idfile parameter can not be unset or empty")
	}

	if err := d.RegistryUpload.Validate(); err != nil {
		return errors.Wrap(err, "") // TODO add section name to error msg
	}

	return nil
}

// Validate validates its content
func (d *DockerImageRegistryUpload) Validate() error {
	if len(d.Repository) == 0 {
		return errors.New("repository parameter can not be unset or empty")
	}

	if len(d.Tag) == 0 {
		return errors.New("tag parameter can not be unset or empty")
	}

	return nil
}

// Validate validates a [[Sources.Files]] section
func (f *FileInputs) Validate() error {
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
