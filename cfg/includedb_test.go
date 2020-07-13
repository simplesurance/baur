package cfg

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pelletier/go-toml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/cfg/resolver"
	"github.com/simplesurance/baur/internal/testutils/fstest"
	"github.com/simplesurance/baur/log"
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

	includeDB := NewIncludeDB(log.StdLogger)
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

	includeDB := NewIncludeDB(log.StdLogger)
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

	includeDB := NewIncludeDB(log.StdLogger)
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

	includeDB := NewIncludeDB(log.StdLogger)
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

	includeDB := NewIncludeDB(log.StdLogger)
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

	includeDB := NewIncludeDB(log.StdLogger)
	loadedIncl, err := includeDB.LoadTaskInclude(
		&resolver.StrReplacement{Old: "$NOTHING"},
		tmpdir,
		filepath.Join(tmpdir, taskInclFilename)+"#"+taskIncl.Task[0].IncludeID,
	)

	require.Error(t, err)
	require.Contains(t, err.Error(), "unique")
	require.Nil(t, loadedIncl)
}
