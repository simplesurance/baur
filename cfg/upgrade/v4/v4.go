package v4

import (
	"fmt"

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

		in := &cfg.InputInclude{IncludeID: NewIncludeID}

		if len(old.BuildInput.GitFiles.Paths) > 0 {
			in.GitFiles = []cfg.GitFileInputs{{Paths: upgradeGitFilePaths(old.BuildInput.GitFiles.Paths)}}
		}

		if len(old.BuildInput.Files.Paths) > 0 {
			in.Files = []cfg.FileInputs{{Paths: old.BuildInput.Files.Paths}}
		}

		if len(old.BuildInput.GolangSources.Environment) > 0 || len(old.BuildInput.GolangSources.Paths) > 0 {
			in.GolangSources = []cfg.GolangSources{
				{
					Environment: old.BuildInput.GolangSources.Environment,
					Queries:     golangSourcesPathsToQuery(old.BuildInput.GolangSources.Paths),
					Tests:       false,
				},
			}
		}

		include.Input = append(include.Input, in)
	}

	if len(old.BuildOutput.DockerImage) > 0 ||
		len(old.BuildOutput.File) > 0 {
		output := cfg.OutputInclude{
			IncludeID: NewIncludeID,
		}

		for _, di := range old.BuildOutput.DockerImage {
			output.DockerImage = append(output.DockerImage, cfg.DockerImageOutput{
				IDFile: di.IDFile,
				RegistryUpload: []cfg.DockerImageRegistryUpload{
					{
						Registry:   di.RegistryUpload.Registry,
						Repository: di.RegistryUpload.Repository,
						Tag:        di.RegistryUpload.Tag,
					},
				},
			},
			)
		}

		for _, f := range old.BuildOutput.File {
			output.File = append(output.File, cfg.FileOutput{
				Path:     f.Path,
				FileCopy: []cfg.FileCopy{{Path: f.FileCopy.Path}},
				S3Upload: []cfg.S3Upload{
					{
						Bucket: f.S3Upload.Bucket,
						Key:    f.S3Upload.DestFile,
					},
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
		p = p + "/..."
		result = append(result, p)
	}

	return result
}

func upgradeGitFilePaths(paths []string) []string {
	result := make([]string, 0, len(paths))

	for _, p := range paths {
		if p == "." {
			result = append(result, "**")
			continue
		}

		result = append(result, p)
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
		Command: []string{"sh", "-c", old.Build.Command},
	}

	if len(old.Build.Input.Files.Paths) > 0 {
		task.Input.Files = []cfg.FileInputs{{Paths: old.Build.Input.Files.Paths}}
	}

	if len(old.Build.Input.GitFiles.Paths) > 0 {
		task.Input.GitFiles = []cfg.GitFileInputs{{Paths: upgradeGitFilePaths(old.Build.Input.GitFiles.Paths)}}
	}

	if len(old.Build.Input.GolangSources.Environment) > 0 || len(old.Build.Input.GolangSources.Paths) > 0 {
		task.Input.GolangSources = []cfg.GolangSources{
			{
				Environment: old.Build.Input.GolangSources.Environment,
				Queries:     golangSourcesPathsToQuery(old.Build.Input.GolangSources.Paths),
				Tests:       false,
			},
		}
	}

	//TODO: dedup code for converting outputs, same code is used used in UpgradeIncludeConfig
	for _, di := range old.Build.Output.DockerImage {
		task.Output.DockerImage = append(task.Output.DockerImage, cfg.DockerImageOutput{
			IDFile: di.IDFile,
			RegistryUpload: []cfg.DockerImageRegistryUpload{
				{
					Registry:   di.RegistryUpload.Registry,
					Repository: di.RegistryUpload.Repository,
					Tag:        di.RegistryUpload.Tag,
				},
			},
		})
	}

	for _, f := range old.Build.Output.File {
		var fc []cfg.FileCopy
		var s3 []cfg.S3Upload

		if f.FileCopy.Path != "" {
			fc = []cfg.FileCopy{{Path: f.FileCopy.Path}}
		}

		if f.S3Upload.Bucket != "" {
			s3 = []cfg.S3Upload{
				{
					Bucket: f.S3Upload.Bucket,
					Key:    f.S3Upload.DestFile,
				},
			}
		}

		task.Output.File = append(task.Output.File, cfg.FileOutput{
			Path:     f.Path,
			FileCopy: fc,
			S3Upload: s3,
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
