package cfg

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/simplesurance/baur/v1/pkg/cfg/resolver"

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
				Command:   []string{"make"},

				Input: Input{
					Files: []FileInputs{{Paths: []string{"*.go"}}},
				},
				Output: Output{
					File: []FileOutput{
						{
							Path:     "a.out",
							FileCopy: []FileCopy{{Path: "/tmp/"}},
						},
					},
				},
			},
		},
	}

	require.NoError(t, taskIncl.Task.validate())

	tmpdir := t.TempDir()

	appCfgPath := filepath.Join(tmpdir, ".app.toml")
	err := app.ToFile(appCfgPath)
	require.NoError(t, err)

	cfgToFile(t, taskIncl, filepath.Join(tmpdir, taskInclFilename))

	loadedApp, err := AppFromFile(appCfgPath)
	require.NoError(t, err)

	includeDB := NewIncludeDB(t.Logf)
	err = loadedApp.Merge(includeDB, &resolver.StrReplacement{Old: "$NOTHING"})
	require.NoError(t, err)

	assert.Error(t, loadedApp.Validate())
}
