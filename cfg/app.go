package cfg

import (
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml"

	"github.com/simplesurance/baur/v1/cfg/resolver"
)

// App stores an application configuration.
type App struct {
	Name     string   `toml:"name" comment:"Name of the application"`
	Includes []string `toml:"includes" comment:"Task-includes that the task inherits.\n Includes are specified in the format <filepath>#<ID>.\n Paths are relative to the application directory.\n. Valid variables: $ROOT."`
	Tasks    Tasks    `toml:"Task"`

	filepath string
}

// ExampleApp returns an exemplary app cfg struct with the name set to the given value.
func ExampleApp(name string) *App {
	return &App{
		Name: name,

		Tasks: []*Task{
			&Task{
				Name:    "build",
				Command: "make dist",
				Input: Input{
					Files: FileInputs{
						Paths: []string{"dbmigrations/*.sql"},
					},
					GitFiles: GitFileInputs{
						Paths: []string{"Makefile"},
					},
					GolangSources: GolangSources{
						Queries:     []string{"./..."},
						Environment: []string{"GOFLAGS=-mod=vendor", "GO111MODULE=on"},
					},
				},
				Output: Output{
					File: []FileOutput{
						{
							Path: "dist/$APPNAME.tar.xz",
							S3Upload: S3Upload{
								Bucket:   "go-artifacts/",
								DestFile: "$APPNAME-$GITCOMMIT.tar.xz",
							},
							FileCopy: FileCopy{
								Path: "/mnt/fileserver/build_artifacts/$APPNAME-$GITCOMMIT.tar.xz",
							},
						},
					},
					DockerImage: []DockerImageOutput{
						{
							IDFile: "$APPNAME-container.id",
							RegistryUpload: DockerImageRegistryUpload{
								Repository: "my-company/$APPNAME",
								Tag:        "$GITCOMMIT",
							},
						},
					},
				},
			},
		},
	}
}

// AppFromFile unmarshals an application configuration from a file and returns
// it.
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

	config.filepath = path

	return &config, err
}

// ToFile marshals the App into toml format and writes it to the given filepath.
func (a *App) ToFile(filepath string, opts ...ToFileOpt) error {
	return toFile(a, filepath, opts...)
}

// FilePath returns the path of the file from which the configuration was loaded.
func (a *App) FilePath() string {
	return a.filepath
}

// Resolve runs the resolvers on string fields that can contain special strings.
// These special strings are replaced with concrete values by the resolvers.
func (a *App) Resolve(resolvers resolver.Resolver) error {
	if err := a.Tasks.Resolve(resolvers); err != nil {
		return FieldErrorWrap(err, "Tasks")
	}

	return nil
}

// Merge merges the configuration with it's includes.
// The task includes listed in App.Includes are loaded via the includedb and
// then appeneded to the task list.
func (a *App) Merge(includedb *IncludeDB, includeSpecResolvers resolver.Resolver) error {
	for _, includeID := range a.Includes {
		include, err := includedb.LoadTaskInclude(includeSpecResolvers, filepath.Dir(a.filepath), includeID)
		if err != nil {
			if errors.Is(err, ErrIncludeIDNotFound) {
				return fmt.Errorf("%s: Task include with given ID not found in include file", includeID)
			}

			return fmt.Errorf("%s: %w", includeID, err)
		}

		a.Tasks = append(a.Tasks, &Task{
			Name:    include.Name,
			Command: include.Command,
			Input:   include.Input,
			Output:  include.Output,
		})
	}

	for _, task := range a.Tasks {
		if err := TaskMerge(task, filepath.Dir(a.filepath), includeSpecResolvers, includedb); err != nil {
			return FieldErrorWrap(err, "Tasks", task.Name)
		}
	}

	return nil
}

// Validate validates the configuration.
// It should be called after Merge().
func (a *App) Validate() error {
	if len(a.Name) == 0 {
		return NewFieldError("can not be empty", "name")
	}

	if strings.Contains(a.Name, ".") {
		return NewFieldError("dots are not allowed in application names", "name")
	}

	if err := validateIncludes(a.Includes); err != nil {
		return FieldErrorWrap(err, "includes")
	}

	if err := a.Tasks.Validate(); err != nil {
		return FieldErrorWrap(err, "Tasks")
	}

	return nil
}
