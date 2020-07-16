package cfg

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/pelletier/go-toml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/cfg/resolver"
	"github.com/simplesurance/baur/internal/testutils/fstest"
)

// cfgToFile marshals a struct to a toml configuration file.
// In opposite to toFile(), no fields are commented in the marshalled versions.
func cfgToFile(t *testing.T, cfg interface{}, path string) {
	t.Helper()

	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}

	encoder := toml.NewEncoder(f)
	encoder.SetTagCommented("false")
	err = encoder.Encode(cfg)
	if err != nil {
		t.Fatal(err)
	}

	err = f.Close()
	if err != nil {
		t.Fatal(err)
	}
}

func inputInclude() InputIncludes {
	return InputIncludes{
		{
			IncludeID: "inputs",
			Files: FileInputs{
				Paths: []string{"Makefile"},
			},

			GitFiles: GitFileInputs{
				Paths: []string{"*.c", "*.h"},
			},

			GolangSources: GolangSources{
				Environment: []string{"GOPATH=."},
				Paths:       []string{"."},
			},
		},
	}
}

func outputInclude() OutputIncludes {
	return OutputIncludes{
		&OutputInclude{
			IncludeID: "outputs",
			DockerImage: []*DockerImageOutput{
				&DockerImageOutput{
					IDFile: "idfile",
					RegistryUpload: DockerImageRegistryUpload{
						Registry:   "localhost:123",
						Repository: "myrepo/calc",
						Tag:        "latest",
					},
				},
			},
			File: []*FileOutput{
				{
					Path: "a.out",
					FileCopy: FileCopy{
						Path: "/tmp/a.out",
					},
					S3Upload: S3Upload{
						Bucket:   "mybucket",
						DestFile: "the-binary",
					},
				},
			},
		},
	}
}

func taskInclude() TaskIncludes {
	return TaskIncludes{
		&TaskInclude{
			IncludeID: "build_task",
			Name:      "build",
			Command:   "make",
		},
	}
}

// TestLoadTaskIncludeWithIncludesInSameFile validates that:
// - TaskIncludes can refer Input/Output includes in the same file.
// - The loaded TaskInclude contains all data from the config.
func TestLoadTaskIncludeWithIncludesInSameFile(t *testing.T) {
	inclFilePath := "include.toml"

	var include Include
	include.Input = inputInclude()
	include.Output = outputInclude()

	include.Task = TaskIncludes{
		&TaskInclude{
			IncludeID: "build_task",
			Name:      "build",
			Command:   "make",
			Includes:  []string{inclFilePath + "#inputs", inclFilePath + "#outputs"},
		},
	}

	tmpdir := fstest.CreateTempDir(t)

	cfgToFile(t, include, filepath.Join(tmpdir, inclFilePath))

	includeDB := NewIncludeDB(t.Logf)
	loadedIncl, err := includeDB.LoadTaskInclude(
		&resolver.StrReplacement{Old: "$NOTHING"},
		tmpdir,
		inclFilePath+"#"+include.Task[0].IncludeID,
	)

	require.NoError(t, err)
	require.NotNil(t, loadedIncl)

	assert.Equal(t, include.Task[0].Name, loadedIncl.Name)
	assert.Equal(t, include.Task[0].IncludeID, loadedIncl.IncludeID)
	assert.Equal(t, include.Task[0].Command, loadedIncl.Command)
	assert.Equal(t, include.Task[0].Includes, loadedIncl.Includes)

	assert.Equal(t, include.Input[0].Files, loadedIncl.Input.Files)
	assert.Equal(t, include.Input[0].GitFiles, loadedIncl.Input.GitFiles)
	assert.Equal(t, include.Input[0].GolangSources, loadedIncl.Input.GolangSources)

	assert.Equal(t, include.Output[0].DockerImage, loadedIncl.Output.DockerImage)
	assert.Equal(t, include.Output[0].File, loadedIncl.Output.File)
}

func TestLoadTaskIncludeWithIncludesInDifferentFiles(t *testing.T) {
	inputInclDir := "input_includes"
	inputInclFilename := filepath.Join(inputInclDir, "inputs.toml")

	var inputIncl Include
	inputIncl.Input = inputInclude()

	outputInclDir := "outputincludes"
	outputInclFilename := filepath.Join(outputInclDir, "inputs.toml")

	var outputIncl Include
	outputIncl.Output = outputInclude()

	taskInclFilename := "tasks.toml"
	taskIncl := Include{
		Task: TaskIncludes{
			&TaskInclude{
				IncludeID: "build_task",
				Name:      "build",
				Command:   "make",
				Includes: []string{
					inputInclFilename + "#" + inputIncl.Input[0].IncludeID,
					outputInclFilename + "#" + outputIncl.Output[0].IncludeID,
				},
			},
		},
	}

	tmpdir := fstest.CreateTempDir(t)

	require.NoError(t, os.MkdirAll(filepath.Join(tmpdir, inputInclDir), 0775))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpdir, outputInclDir), 0775))

	cfgToFile(t, inputIncl, filepath.Join(tmpdir, inputInclFilename))
	cfgToFile(t, outputIncl, filepath.Join(tmpdir, outputInclFilename))
	cfgToFile(t, taskIncl, filepath.Join(tmpdir, taskInclFilename))

	includeDB := NewIncludeDB(t.Logf)
	loadedIncl, err := includeDB.LoadTaskInclude(
		&resolver.StrReplacement{Old: "$NOTHING"},
		tmpdir,
		filepath.Join(tmpdir, taskInclFilename)+"#"+taskIncl.Task[0].IncludeID,
	)

	require.NoError(t, err)
	require.NotNil(t, loadedIncl)
}

// TestIncludePathsAreRelativeToCfg ensures that the paths in the Includes list
// of a TaskInclude are relative to the TaskInclude file.
func TestIncludePathsAreRelativeToCfg(t *testing.T) {
	inputInclDirName := "subdir"
	inputInclFilename := "inputs.toml"

	var inputIncl Include
	inputIncl.Input = inputInclude()

	taskInclFilename := "tasks.toml"
	taskIncl := Include{
		Task: TaskIncludes{
			&TaskInclude{
				IncludeID: "build_task",
				Name:      "build",
				Command:   "make",
				Includes: []string{
					filepath.Join(inputInclDirName, inputInclFilename) + "#" + inputIncl.Input[0].IncludeID,
				},
			},
		},
	}

	tmpdir := fstest.CreateTempDir(t)

	require.NoError(t, os.MkdirAll(filepath.Join(tmpdir, inputInclDirName), 0775))

	cfgToFile(t, inputIncl, filepath.Join(tmpdir, "subdir", inputInclFilename))
	cfgToFile(t, taskIncl, filepath.Join(tmpdir, taskInclFilename))

	includeDB := NewIncludeDB(t.Logf)
	loadedIncl, err := includeDB.LoadTaskInclude(
		&resolver.StrReplacement{Old: "$NOTHING"},
		tmpdir,
		filepath.Join(tmpdir, taskInclFilename)+"#"+taskIncl.Task[0].IncludeID,
	)

	require.NoError(t, err)
	require.NotNil(t, loadedIncl)
}

func TestAbsIncludePathsFail(t *testing.T) {
	var inputIncl Include
	inputIncl.Input = inputInclude()

	tmpdir := fstest.CreateTempDir(t)

	absInputInclPath := filepath.Join(tmpdir, "inputs.toml")
	taskIncl := Include{
		Task: TaskIncludes{
			&TaskInclude{
				IncludeID: "build_task",
				Name:      "build",
				Command:   "make",
				Includes: []string{
					absInputInclPath + "#" + inputIncl.Input[0].IncludeID,
				},
			},
		},
	}

	taskInclFilename := "tasks.toml"

	cfgToFile(t, inputIncl, absInputInclPath)
	cfgToFile(t, taskIncl, filepath.Join(tmpdir, taskInclFilename))

	includeDB := NewIncludeDB(t.Logf)
	loadedIncl, err := includeDB.LoadTaskInclude(
		&resolver.StrReplacement{Old: "$NOTHING"},
		tmpdir,
		taskInclFilename+"#"+taskIncl.Task[0].IncludeID,
	)

	require.Error(t, err)
	require.Contains(t, err.Error(), "absolute")
	require.Nil(t, loadedIncl)
}

func TestEnsureInputIncludeIDsMustBeUnique(t *testing.T) {
	var inputIncl Include
	inputIncl.Input = inputInclude()
	inputIncl.Input = append(inputIncl.Input, inputInclude()...)
	inputInclFilename := "inputs.toml"

	var taskIncl Include
	taskInclFilename := "tasks.toml"
	taskIncl.Task = taskInclude()
	taskIncl.Task[0].Includes = []string{inputInclFilename + "#" + inputIncl.Input[0].IncludeID}

	tmpdir := fstest.CreateTempDir(t)

	cfgToFile(t, inputIncl, filepath.Join(tmpdir, inputInclFilename))
	cfgToFile(t, taskIncl, filepath.Join(tmpdir, taskInclFilename))

	includeDB := NewIncludeDB(t.Logf)
	loadedIncl, err := includeDB.LoadTaskInclude(
		&resolver.StrReplacement{Old: "$NOTHING"},
		tmpdir,
		filepath.Join(tmpdir, taskInclFilename)+"#"+taskIncl.Task[0].IncludeID,
	)

	require.Error(t, err)
	require.Contains(t, err.Error(), "unique")
	require.Nil(t, loadedIncl)

}

func TestEnsureOutputIncludeIDsMustBeUnique(t *testing.T) {
	var outputIncl Include
	outputIncl.Output = outputInclude()
	outputIncl.Output = append(outputIncl.Output, outputInclude()...)
	outputInclFilename := "outputs.toml"

	var taskIncl Include
	taskInclFilename := "tasks.toml"
	taskIncl.Task = taskInclude()
	taskIncl.Task[0].Includes = []string{outputInclFilename + "#" + outputIncl.Output[0].IncludeID}

	tmpdir := fstest.CreateTempDir(t)

	cfgToFile(t, outputIncl, filepath.Join(tmpdir, outputInclFilename))
	cfgToFile(t, taskIncl, filepath.Join(tmpdir, taskInclFilename))

	includeDB := NewIncludeDB(t.Logf)
	loadedIncl, err := includeDB.LoadTaskInclude(
		&resolver.StrReplacement{Old: "$NOTHING"},
		tmpdir,
		filepath.Join(tmpdir, taskInclFilename)+"#"+taskIncl.Task[0].IncludeID,
	)

	require.Error(t, err)
	require.Contains(t, err.Error(), "unique")
	require.Nil(t, loadedIncl)
}

// TestTestTaskInclude ensures includes specified in Task.Includes are included.
func TestTaskInclude(t *testing.T) {
	type testcaseIncludeConfigs struct {
		filename string
		cfg      Include
	}

	testcases := []struct {
		name string
		// The current validation logic in the testcase requires that
		// the apppAconfig only contains a single task definition
		appConfig     App
		includeConfig testcaseIncludeConfigs
	}{
		{
			name: "multipleIncludes",
			appConfig: App{
				Name: "testapp",
				Tasks: Tasks{
					{
						Name:    "build",
						Command: "make",
						Includes: []string{
							"include.toml#input",
							"include.toml#input2",
							"include.toml#output",
							"include.toml#output2",
						},
					},
				},
			},
			includeConfig: testcaseIncludeConfigs{
				filename: "include.toml",
				cfg: Include{
					Input: InputIncludes{
						{
							IncludeID: "input",
							Files: FileInputs{
								Paths: []string{"*.go", "*.sh", "*.bat"},
							},
							GitFiles: GitFileInputs{
								Paths: []string{"*.txt", "*.d"},
							},
							GolangSources: GolangSources{
								Environment: []string{"A=B"},
								Paths:       []string{"."},
							},
						},
						{
							IncludeID: "input2",
							Files: FileInputs{
								Paths: []string{"*.c", "*.sh"},
							},
							GitFiles: GitFileInputs{
								Paths: []string{"*.txt", "hellofile"},
							},
							GolangSources: GolangSources{
								Environment: []string{"C=D"},
								Paths:       []string{"cmd/"},
							},
						},
					},

					Output: OutputIncludes{
						{
							IncludeID: "output",
							DockerImage: []*DockerImageOutput{
								{
									IDFile: "idfile",
									RegistryUpload: DockerImageRegistryUpload{
										Registry:   "registry",
										Repository: "repo",
										Tag:        "tag",
									},
								},
							},
							File: []*FileOutput{
								{
									Path:     "path",
									FileCopy: FileCopy{Path: "/tmp/"},
									S3Upload: S3Upload{
										Bucket:   "bucket",
										DestFile: "dest",
									},
								},
							},
						},
						{
							IncludeID: "output2",
							DockerImage: []*DockerImageOutput{
								{
									IDFile: "idfile1",
									RegistryUpload: DockerImageRegistryUpload{
										Registry:   "registry1",
										Repository: "repo1",
										Tag:        "tag",
									},
								},
							},
							File: []*FileOutput{
								{
									Path:     "path",
									FileCopy: FileCopy{Path: "/data/"},
									S3Upload: S3Upload{
										Bucket:   "bucket1",
										DestFile: "dest1",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			tmpdir := fstest.CreateTempDir(t)

			appCfgPath := filepath.Join(tmpdir, ".app.toml")
			require.NoError(t, tc.appConfig.ToFile(appCfgPath))

			includeCfgPath := filepath.Join(tmpdir, tc.includeConfig.filename)
			require.NoError(t, tc.includeConfig.cfg.ToFile(includeCfgPath))

			loadedApp, err := AppFromFile(appCfgPath)
			require.NoError(t, err)

			includeDB := NewIncludeDB(t.Logf)
			err = loadedApp.Merge(includeDB, &resolver.StrReplacement{Old: "$NOTHING"})
			require.NoError(t, err)

			assert.Equal(t, tc.appConfig.Name, loadedApp.Name)

			// current validation logic also only works with exactly 1 task
			require.Len(t, loadedApp.Tasks, 1)
			loadedTask := loadedApp.Tasks[0]

			for _, inputIncl := range tc.includeConfig.cfg.Input {
				for _, path := range inputIncl.FileInputs().Paths {
					assert.Contains(t, loadedTask.Input.FileInputs().Paths, path, "FileInput path missing")
				}

				for _, path := range inputIncl.GitFileInputs().Paths {
					assert.Contains(t, loadedTask.Input.GitFileInputs().Paths, path, "GitFileInput missing")
				}

				// related bug: https://github.com/simplesurance/baur/issues/169
				for _, env := range loadedTask.Input.GolangSources.Environment {
					assert.Contains(t, loadedTask.Input.GolangSourcesInputs().Environment, env, "GolangSources env missing")
				}

				for _, path := range loadedTask.Input.GolangSources.Paths {
					assert.Contains(t, loadedTask.Input.GolangSourcesInputs().Paths, path, "GolangSources path missing")
				}
			}

			for _, outputIncl := range tc.includeConfig.cfg.Output {
				for _, di := range outputIncl.DockerImage {
					assert.Contains(t, loadedTask.Output.DockerImage, di)
				}

				for _, fileOutput := range outputIncl.File {
					assert.Contains(t, loadedTask.Output.File, fileOutput)
				}
			}

		})
	}
}

func TestTaskIncludeFailsForNonExistingIncludeFile(t *testing.T) {
	app := App{
		Name: "testapp",
		Tasks: Tasks{
			{
				Name:    "build",
				Command: "make",
				Includes: []string{
					"include.toml#input",
				},
			},
		},
	}

	tmpdir := fstest.CreateTempDir(t)
	appCfgPath := filepath.Join(tmpdir, ".app.toml")
	require.NoError(t, app.ToFile(appCfgPath))

	loadedApp, err := AppFromFile(appCfgPath)
	require.NoError(t, err)

	includeDB := NewIncludeDB(t.Logf)
	err = loadedApp.Merge(includeDB, &resolver.StrReplacement{Old: "$NOTHING"})
	require.True(
		t,
		os.IsNotExist(err) || os.IsNotExist(errors.Unwrap(err)),
		"merge did not return NotExist error: %v", err,
	)
}

func TestTaskIncludeFailsForNonExistingIncludeName(t *testing.T) {
	app := App{
		Name: "testapp",
		Tasks: Tasks{
			{
				Name:    "build",
				Command: "make",
				Includes: []string{
					"include.toml#nonexisting",
				},
			},
		},
	}

	include := Include{
		Input: InputIncludes{
			{
				IncludeID: "input",
				Files: FileInputs{
					Paths: []string{"*.go", "*.sh", "*.bat"},
				},
			},
		},

		Output: OutputIncludes{
			{
				IncludeID: "output",
				File: []*FileOutput{
					{
						Path:     "path",
						FileCopy: FileCopy{Path: "/tmp/"},
					},
				},
			},
		},
	}

	tmpdir := fstest.CreateTempDir(t)
	appCfgPath := filepath.Join(tmpdir, ".app.toml")
	require.NoError(t, app.ToFile(appCfgPath))

	require.NoError(t, include.ToFile(filepath.Join(tmpdir, "include.toml")))

	loadedApp, err := AppFromFile(appCfgPath)
	require.NoError(t, err)

	includeDB := NewIncludeDB(t.Logf)
	err = loadedApp.Merge(includeDB, &resolver.StrReplacement{Old: "$NOTHING"})
	require.True(t, errors.Is(err, ErrIncludeIDNotFound), "merge did not return ErrIncludeIDNotFound: %v", err)
}
