package v4

import (
	"fmt"
	"path/filepath"

	cfgv0 "github.com/simplesurance/baur/cfg"
	"github.com/simplesurance/baur/v1/cfg"
)

const NewIncludeID = "main"

func UpgradeRepositoryConfig(old *cfgv0.Repository) *cfg.Repository {
	return &cfg.Repository{
		ConfigVersion: cfg.Version,
		Database: cfg.Database{
			PGSQLURL: old.Database.PGSQLURL,
		},
		Discover: cfg.Discover{
			Dirs:        old.Discover.Dirs,
			SearchDepth: old.Discover.SearchDepth,
		},
	}
}

func UpgradeIncludeConfig(old *cfgv0.Include) *cfg.Include {
	include := &cfg.Include{}

	if len(old.BuildInput.Files.Paths) > 0 ||
		len(old.BuildInput.GitFiles.Paths) > 0 ||
		len(old.BuildInput.GolangSources.Environment) > 0 ||
		len(old.BuildInput.GolangSources.Paths) > 0 {

		include.Input = append(include.Input, &cfg.InputInclude{
			IncludeID: NewIncludeID,
			Files: cfg.FileInputs{
				Paths: old.BuildInput.Files.Paths,
			},

			GitFiles: cfg.GitFileInputs{
				Paths: old.BuildInput.GitFiles.Paths,
			},
			GolangSources: cfg.GolangSources{
				Environment: old.BuildInput.GolangSources.Environment,
				Queries:     golangSourcesPathsToQuery(old.BuildInput.GolangSources.Paths),
			},
		})
	}

	if len(old.BuildOutput.DockerImage) > 0 ||
		len(old.BuildOutput.File) > 0 {
		output := cfg.OutputInclude{
			IncludeID: NewIncludeID,
		}

		for _, di := range old.BuildOutput.DockerImage {
			output.DockerImage = append(output.DockerImage, cfg.DockerImageOutput{
				IDFile: di.IDFile,
				RegistryUpload: cfg.DockerImageRegistryUpload{
					Registry:   di.RegistryUpload.Registry,
					Repository: di.RegistryUpload.Repository,
					Tag:        di.RegistryUpload.Tag,
				},
			})
		}

		for _, f := range old.BuildOutput.File {
			output.File = append(output.File, cfg.FileOutput{
				Path:     f.Path,
				FileCopy: cfg.FileCopy{Path: f.FileCopy.Path},
				S3Upload: cfg.S3Upload{
					Bucket:   f.S3Upload.Bucket,
					DestFile: f.S3Upload.DestFile,
				},
			})
		}

		include.Output = append(include.Output, &output)
	}

	return include
}

func golangSourcesPathsToQuery(paths []string) []string {
	result := make([]string, 0, len(paths))
	for _, p := range paths {
		q := filepath.Join(".", p, "./...")
		result = append(result, q)
	}

	return result
}

// UpgradeAppConfig converts a version 4 app config to version 5.
// Includes are not upgrades, NewIncludeID is appened to include references.
func UpgradeAppConfig(old *cfgv0.App) *cfg.App {
	if old.Build.Command == "" {
		return &cfg.App{Name: old.Name}
	}

	task := cfg.Task{
		Name:    "build",
		Command: old.Build.Command,
	}

	task.Input.Files.Paths = old.Build.Input.Files.Paths
	task.Input.GitFiles.Paths = old.Build.Input.GitFiles.Paths
	task.Input.GolangSources.Environment = old.Build.Input.GolangSources.Environment
	task.Input.GolangSources.Queries = golangSourcesPathsToQuery(old.Build.Input.GolangSources.Paths)

	//TODO: dedup code for converting outputs, same code is used used in UpgradeIncludeConfig
	for _, di := range old.Build.Output.DockerImage {
		task.Output.DockerImage = append(task.Output.DockerImage, cfg.DockerImageOutput{
			IDFile: di.IDFile,
			RegistryUpload: cfg.DockerImageRegistryUpload{
				Registry:   di.RegistryUpload.Registry,
				Repository: di.RegistryUpload.Repository,
				Tag:        di.RegistryUpload.Tag,
			},
		})
	}

	for _, f := range old.Build.Output.File {
		task.Output.File = append(task.Output.File, cfg.FileOutput{
			Path:     f.Path,
			FileCopy: cfg.FileCopy{Path: f.FileCopy.Path},
			S3Upload: cfg.S3Upload{
				Bucket:   f.S3Upload.Bucket,
				DestFile: f.S3Upload.DestFile,
			},
		})
	}

	for _, includePath := range old.Build.Includes {
		task.Includes = append(task.Includes,
			fmt.Sprintf("%s#%s", includePath, NewIncludeID),
		)
	}

	return &cfg.App{
		Name:  old.Name,
		Tasks: cfg.Tasks{&task},
	}
}
