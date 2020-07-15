package cfg

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/simplesurance/baur/cfg/resolver"
	"github.com/simplesurance/baur/internal/testutils/fstest"
	"github.com/simplesurance/baur/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ExampleApp_WrittenAndReadCfgIsValid(t *testing.T) {
	tmpfileFD, err := ioutil.TempFile("", "baur")
	if err != nil {
		t.Fatal("opening tmpfile failed: ", err)
	}

	tmpfileName := tmpfileFD.Name()
	tmpfileFD.Close()
	os.Remove(tmpfileName)

	a := ExampleApp("shop")
	if err := a.Validate(); err != nil {
		t.Error("example conf fails validation: ", err)
	}

	if err := a.ToFile(tmpfileName); err != nil {
		t.Fatal("writing conf to file failed: ", err)
	}

	rRead, err := AppFromFile(tmpfileName)
	if err != nil {
		t.Fatal("reading conf from file failed: ", err)
	}

	if err := rRead.Validate(); err != nil {
		t.Errorf("validating conf from file failed: %s\nFile Content: %+v", err, rRead)
	}
}

func TestEnsureValidateFailsOnDuplicateTaskNames(t *testing.T) {
	taskInclFilename := "tasks.toml"

	app := ExampleApp("testapp")
	taskName := app.Tasks[0].Name
	app.Includes = []string{taskInclFilename + "#" + taskName}

	require.NoError(t, app.Validate())

	taskIncl := Include{
		Task: TaskIncludes{
			&TaskInclude{
				IncludeID: taskName,
				Name:      taskName,
				Command:   "make",

				Input: Input{
					Files: FileInputs{Paths: []string{"*.go"}},
				},
				Output: Output{
					File: []*FileOutput{
						{
							Path:     "a.out",
							FileCopy: FileCopy{Path: "/tmp/"},
						},
					},
				},
			},
		},
	}

	require.NoError(t, taskIncl.Task.Validate())

	tmpdir := fstest.CreateTempDir(t)

	appCfgPath := filepath.Join(tmpdir, ".app.toml")
	err := app.ToFile(appCfgPath)
	require.NoError(t, err)

	cfgToFile(t, taskIncl, filepath.Join(tmpdir, taskInclFilename))

	loadedApp, err := AppFromFile(appCfgPath)
	require.NoError(t, err)

	includeDB := NewIncludeDB(log.StdLogger)
	err = loadedApp.Merge(includeDB, &resolver.StrReplacement{Old: "$NOTHING"})
	require.NoError(t, err)

	assert.Error(t, loadedApp.Validate())
}

// TODO: add test that ensures multiple inputincludes/outputincludes elements and multiple include files are handled correctly
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

			includeDB := NewIncludeDB(log.StdLogger)
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

	includeDB := NewIncludeDB(log.StdLogger)
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

	includeDB := NewIncludeDB(log.StdLogger)
	err = loadedApp.Merge(includeDB, &resolver.StrReplacement{Old: "$NOTHING"})
	require.True(t, errors.Is(err, ErrIncludeIDNotFound), "merge did not return ErrIncludeIDNotFound: %v", err)
}
