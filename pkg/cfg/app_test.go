package cfg

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/simplesurance/baur/v3/internal/prettyprint"

	"github.com/stretchr/testify/require"
)

func Test_ExampleApp_WrittenAndReadCfgIsValid(t *testing.T) {
	tmpfileFD, err := os.CreateTemp("", "baur")
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
		t.Errorf("validating conf after writing and reading it again from file failed: %s\nFile Content: %+v", err, prettyprint.AsString(rRead))
	}
}

func TestAppNameValidation(t *testing.T) {
	testcases := []struct {
		AppName        string
		ExpectedErrStr string
	}{
		{
			AppName:        "sh.o.p",
			ExpectedErrStr: "character not allowed",
		},
		{
			AppName:        "sh##",
			ExpectedErrStr: "character not allowed",
		},
		{
			AppName:        "star***",
			ExpectedErrStr: "character not allowed",
		},
	}
	for _, tc := range testcases {
		t.Run(tc.AppName, func(t *testing.T) {
			a := ExampleApp(tc.AppName)
			err := a.Validate()
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.ExpectedErrStr)
		})
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
	err = loadedApp.Merge(includeDB, &mockResolver{})
	require.NoError(t, err)

	require.Error(t, loadedApp.Validate())
}
