package cfg

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml"
)

// App stores an application configuration.
type App struct {
	Name     string   `toml:"name" comment:"Application name"`
	Includes []string `toml:"includes" comment:"Task-includes that the task inherits.\n Includes are specified in the format FILEPATH#INCLUDE_ID.\n Paths are relative to the application directory."`
	Tasks    Tasks    `toml:"Task"`

	filepath string
}

// ExampleApp returns an exemplary app cfg struct with the name set to the given value.
func ExampleApp(name string) *App {
	return &App{
		Name: name,
		Tasks: []*Task{
			{
				Name:    "build",
				Command: []string{"make", "dist"},
				Input: Input{
					Files: []FileInputs{
						{
							Paths: []string{"dbmigrations/*.sql"},
						},
					},
					ExcludedFiles: FileExcludeList{
						Paths: []string{"dbmigrations/1.sql"},
					},
					EnvironmentVariables: []EnvVarsInputs{
						{
							Names:    []string{"APP_VERSION", name + "_*"},
							Optional: false,
						},
					},
					GolangSources: []GolangSources{
						{
							Queries:     []string{"./..."},
							Environment: []string{"GOFLAGS=-mod=vendor", "GO111MODULE=on"},
							BuildFlags:  []string{"-tags=linux"},
						},
					},
				},
				Output: Output{
					File: []FileOutput{
						{
							Path: "dist/{{ .appname }}.tar.xz",
							S3Upload: []S3Upload{
								{
									Bucket: "go-artifacts/",
									Key:    "{{ .AppName }}-{{ gitCommit }}.tar.xz",
								},
							},
							FileCopy: []FileCopy{
								{
									Path: "/mnt/fileserver/build_artifacts/{{ .AppName }}-{{ gitCommit }}.tar.xz",
								},
							},
						},
					},
					DockerImage: []DockerImageOutput{
						{
							IDFile: "{{ .appname }}-container.id",
							RegistryUpload: []DockerImageRegistryUpload{
								{
									Repository: "my-company/{{ .AppName }}",
									Tag:        "{{ ENV BRANCH_NAME }}-{{ gitCommit }}",
								},
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

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	err = toml.Unmarshal(content, &config)
	if err != nil {
		return nil, err
	}

	config.filepath = path

	for _, task := range config.Tasks {
		task.cfgFiles = map[string]struct{}{config.filepath: {}}
	}

	return &config, err
}

// ToFile marshals the App into toml format and writes it to the given filepath.
func (a *App) ToFile(filepath string, opts ...toFileOpt) error {
	a.filepath = filepath
	return toFile(a, filepath, opts...)
}

// FilePath returns the path of the app configuration file
func (a *App) FilePath() string {
	return a.filepath
}

// Resolve runs the resolvers on string fields that can contain special strings.
// These special strings are replaced with concrete values by the resolvers.
func (a *App) Resolve(resolver Resolver) error {
	if err := a.Tasks.resolve(resolver); err != nil {
		return fieldErrorWrap(err, "Tasks")
	}

	return nil
}

// Merge merges the configuration with it's includes.
// The task includes listed in App.Includes are loaded via the includedb and
// then appeneded to the task list.
func (a *App) Merge(includedb *IncludeDB, includeSpecResolver Resolver) error {
	for _, includeID := range a.Includes {
		taskInclude, err := includedb.loadTaskInclude(includeSpecResolver, filepath.Dir(a.filepath), includeID)
		if err != nil {
			return fmt.Errorf("%s: %w", includeID, err)
		}

		task := taskInclude.toTask()
		task.addCfgFilepath(a.filepath)
		a.Tasks = append(a.Tasks, task)
	}

	for _, task := range a.Tasks {
		err := taskMerge(task, filepath.Dir(a.filepath), includeSpecResolver, includedb)
		if err != nil {
			return fieldErrorWrap(err, "Tasks", task.Name)
		}
	}

	return nil
}

// Validate validates the configuration.
// It should be called after Merge().
func (a *App) Validate() error {
	if err := validateTaskOrAppName(a.Name); err != nil {
		return fieldErrorWrap(err, "name")
	}

	if strings.Contains(a.Name, ".") {
		return newFieldError("dots are not allowed in application names", "name")
	}

	if err := validateIncludes(a.Includes); err != nil {
		return fieldErrorWrap(err, "includes")
	}

	if err := a.Tasks.validate(); err != nil {
		return fieldErrorWrap(err, "Tasks")
	}

	return nil
}
