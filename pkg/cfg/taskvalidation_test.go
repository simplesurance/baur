package cfg

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validTask() *Task {
	return &Task{
		Name:    "build",
		Command: []string{"make"},
		Input:   Input{Files: []FileInputs{{Paths: []string{"a"}}}},
	}
}

func TestTaskValidate(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*Task)
		wantErr string
	}{
		{"valid", func(*Task) {}, ""},
		{"empty command", func(tk *Task) { tk.Command = nil }, "command: can not be empty"},
		{"empty name", func(tk *Task) { tk.Name = "" }, "name: can not be empty"},
		{"dot in name", func(tk *Task) { tk.Name = "a.b" }, "name"},
		{"leading space in name", func(tk *Task) { tk.Name = " a" }, "leading or trailing"},
		{"absolute include", func(tk *Task) { tk.Includes = []string{filepath.FromSlash("/abs/i.toml#x")} }, "includes"},
		{"invalid input", func(tk *Task) { tk.Input.Files = []FileInputs{{Paths: []string{""}}} }, "Input.Files"},
		{"invalid output", func(tk *Task) { tk.Output.File = []FileOutput{{}} }, "Output.File"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tk := validTask()
			tc.modify(tk)

			err := taskValidate(tk)

			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}

			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}

func TestTasksValidateReportsTaskName(t *testing.T) {
	broken := validTask()
	broken.Name = "broken"
	broken.Command = nil

	err := Tasks{validTask(), broken}.validate()

	require.ErrorContains(t, err, "Task.broken")
}

func TestTasksValidateRejectsTasksWithoutNameWithoutNamePrefix(t *testing.T) {
	tk := validTask()
	tk.Name = ""

	err := Tasks{tk}.validate()

	require.ErrorContains(t, err, "Task.name")
}

func TestTasksValidateDuplicateNames(t *testing.T) {
	err := Tasks{validTask(), validTask()}.validate()

	require.ErrorContains(t, err, "multiple tasks with name 'build'")
}

func TestTasksValidateTaskInfoReferencesUnknownTask(t *testing.T) {
	tk := validTask()
	tk.Input.TaskInfos = []TaskInfo{{TaskName: "ghost", EnvVarName: "X"}}

	err := Tasks{tk}.validate()

	require.ErrorContains(t, err, `a task named "ghost" does not exists`)
}

func TestTaskAddCfgFilepathAndFilepaths(t *testing.T) {
	tk := &Task{cfgFiles: map[string]struct{}{"/a": {}}}

	tk.addCfgFilepath("/b")
	tk.addCfgFilepath("/b")

	assert.ElementsMatch(t, []string{"/a", "/b"}, tk.Filepaths())
	assert.Panics(t, func() { tk.addCfgFilepath("") })
}

func TestTaskResolveResolvesCommandInputAndOutput(t *testing.T) {
	tk := &Task{
		Command: []string{"make", "x"},
		Input:   Input{Files: []FileInputs{{Paths: []string{"in"}}}},
		Output:  Output{File: []FileOutput{{Path: "out"}}},
	}

	require.NoError(t, tk.resolve(upperResolver))

	assert.Equal(t, []string{"MAKE", "X"}, tk.Command)
	assert.Equal(t, []string{"IN"}, tk.Input.Files[0].Paths)
	assert.Equal(t, "OUT", tk.Output.File[0].Path)
}

func TestTaskResolveErrors(t *testing.T) {
	r := failingResolverOn("bad")

	require.ErrorContains(t, (&Task{Command: []string{"bad"}}).resolve(r), "Command")
	require.ErrorContains(t,
		(&Task{Input: Input{Files: []FileInputs{{Paths: []string{"bad"}}}}}).resolve(r), "Input")
	require.ErrorContains(t,
		(&Task{Output: Output{File: []FileOutput{{Path: "bad"}}}}).resolve(r), "Output")

	err := Tasks{{Name: "t1", Command: []string{"bad"}}}.resolve(r)
	require.ErrorContains(t, err, "Tasks.t1")
}

func TestTaskIncludeToTaskIsIndependentCopy(t *testing.T) {
	incl := &TaskInclude{
		IncludeID: "id",
		Name:      "build",
		Command:   []string{"make"},
		Input:     Input{Files: []FileInputs{{Paths: []string{"a"}}}},
		Output:    Output{File: []FileOutput{{Path: "o"}}},
		cfgFiles:  map[string]struct{}{"/inc.toml": {}},
	}

	tk := incl.toTask()
	tk.Command[0] = "changed"
	tk.Input.Files[0].Paths[0] = "changed"
	tk.Output.File[0].Path = "changed"
	tk.cfgFiles["/other"] = struct{}{}

	assert.Equal(t, "build", tk.Name)
	assert.Equal(t, "make", incl.Command[0])
	assert.Equal(t, "a", incl.Input.Files[0].Paths[0])
	assert.Equal(t, "o", incl.Output.File[0].Path)
	assert.Len(t, incl.cfgFiles, 1)
}

func TestTaskIncludeValidate(t *testing.T) {
	valid := func() *TaskInclude {
		return &TaskInclude{
			IncludeID: "id", Name: "build", Command: []string{"make"},
			Input: Input{Files: []FileInputs{{Paths: []string{"a"}}}},
		}
	}

	require.NoError(t, valid().validate())

	tk := valid()
	tk.IncludeID = ""
	require.ErrorContains(t, tk.validate(), "is empty")

	tk = valid()
	tk.IncludeID = "a#b"
	require.ErrorContains(t, tk.validate(), "a#b")

	tk = valid()
	tk.Command = nil
	require.ErrorContains(t, tk.validate(), "command")
}

func TestTaskMergeMergesInputAndOutputIncludesAndTracksFiles(t *testing.T) {
	dir := t.TempDir()
	incPath := filepath.Join(dir, "inc.toml")
	cfgToFile(t, Include{Input: inputInclude(), Output: outputInclude()}, incPath)

	tk := &Task{
		Name:     "build",
		Command:  []string{"make"},
		Includes: []string{"inc.toml#inputs", "inc.toml#outputs"},
		Input:    Input{Files: []FileInputs{{Paths: []string{"own"}}}},
		cfgFiles: map[string]struct{}{"/app.toml": {}},
	}

	require.NoError(t, taskMerge(tk, dir, &mockResolver{}, NewIncludeDB(nil)))

	require.Len(t, tk.Input.Files, 3, "own input plus the 2 inputs of the include")
	assert.Equal(t, []string{"own"}, tk.Input.Files[0].Paths)
	assert.Len(t, tk.Input.GolangSources, 1)
	assert.Len(t, tk.Output.File, 1)
	assert.Len(t, tk.Output.DockerImage, 1)
	assert.ElementsMatch(t, []string{"/app.toml", incPath}, tk.Filepaths())
}

func TestTaskMergeErrors(t *testing.T) {
	dir := t.TempDir()
	cfgToFile(t, Include{Input: inputInclude()}, filepath.Join(dir, "inc.toml"))

	t.Run("unknown include id", func(t *testing.T) {
		tk := &Task{Includes: []string{"inc.toml#nope"}, cfgFiles: map[string]struct{}{}}

		err := taskMerge(tk, dir, &mockResolver{}, NewIncludeDB(nil))

		require.ErrorIs(t, err, ErrIncludeIDNotFound)
		assert.Contains(t, err.Error(), "inc.toml#nope")
	})

	t.Run("missing include file", func(t *testing.T) {
		tk := &Task{Includes: []string{"missing.toml#x"}, cfgFiles: map[string]struct{}{}}

		err := taskMerge(tk, dir, &mockResolver{}, NewIncludeDB(nil))

		require.ErrorIs(t, err, os.ErrNotExist)
	})
}

func TestAppFromFile(t *testing.T) {
	dir := t.TempDir()

	t.Run("sets file path on app and tasks", func(t *testing.T) {
		p := filepath.Join(dir, ".app.toml")
		require.NoError(t, os.WriteFile(p, []byte(`
name = "myapp"
[[Task]]
name = "build"
command = ["make"]
[Task.Input]
[[Task.Input.Files]]
paths = ["a"]
`), 0o600))

		app, err := AppFromFile(p)

		require.NoError(t, err)
		assert.Equal(t, "myapp", app.Name)
		assert.Equal(t, p, app.FilePath())
		require.Len(t, app.Tasks, 1)
		assert.Equal(t, []string{p}, app.Tasks[0].Filepaths())
		require.NoError(t, app.Validate())
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := AppFromFile(filepath.Join(dir, "missing.toml"))

		require.ErrorIs(t, err, os.ErrNotExist)
	})

	t.Run("invalid toml", func(t *testing.T) {
		p := filepath.Join(dir, "bad.toml")
		require.NoError(t, os.WriteFile(p, []byte("name = = x"), 0o600))

		_, err := AppFromFile(p)

		require.Error(t, err)
	})
}

func TestAppValidate(t *testing.T) {
	tests := []struct {
		name    string
		app     App
		wantErr string
	}{
		{"valid without tasks", App{Name: "a"}, ""},
		{"empty name", App{}, "name: can not be empty"},
		{"dot in name", App{Name: "a.b"}, "name"},
		{"forbidden char in name", App{Name: "a*"}, "'*' character not allowed"},
		{"absolute include", App{Name: "a", Includes: []string{filepath.FromSlash("/x/i.toml") + "#t"}}, "includes"},
		{"invalid task", App{Name: "a", Tasks: Tasks{{Name: "t"}}}, "Tasks.Task.t"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.app.Validate()

			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}

			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}

func TestAppResolveErrorContainsTaskPath(t *testing.T) {
	app := App{Tasks: Tasks{{Name: "t1", Command: []string{"bad"}}}}

	err := app.Resolve(failingResolverOn("bad"))

	require.ErrorContains(t, err, "Tasks.Tasks.t1")
}

func TestAppMergeAppendsIncludedTasksAndMergesTaskIncludes(t *testing.T) {
	dir := t.TempDir()
	incPath := filepath.Join(dir, "inc.toml")
	cfgToFile(t, Include{Input: inputInclude(), Task: taskInclude()}, incPath)

	appPath := filepath.Join(dir, ".app.toml")
	app := &App{
		Name:     "app",
		Includes: []string{"inc.toml#build_task"},
		Tasks: Tasks{{
			Name:     "own",
			Command:  []string{"x"},
			Includes: []string{"inc.toml#inputs"},
			cfgFiles: map[string]struct{}{appPath: {}},
		}},
		filepath: appPath,
	}

	require.NoError(t, app.Merge(NewIncludeDB(nil), &mockResolver{}))

	require.Len(t, app.Tasks, 2)
	assert.Equal(t, "own", app.Tasks[0].Name)
	assert.Equal(t, "build", app.Tasks[1].Name, "included tasks are appended")
	assert.ElementsMatch(t, []string{appPath, incPath}, app.Tasks[1].Filepaths())
	assert.Len(t, app.Tasks[0].Input.Files, 2, "inputs of the include are merged into the task")
}

func TestAppMergeErrors(t *testing.T) {
	dir := t.TempDir()
	app := &App{Includes: []string{"missing.toml#x"}, filepath: filepath.Join(dir, ".app.toml")}

	err := app.Merge(NewIncludeDB(nil), &mockResolver{})

	require.ErrorIs(t, err, os.ErrNotExist)
	assert.Contains(t, err.Error(), "missing.toml#x")

	app = &App{
		Tasks:    Tasks{{Name: "t", Includes: []string{"inc.toml#nope"}, cfgFiles: map[string]struct{}{}}},
		filepath: filepath.Join(dir, ".app.toml"),
	}
	cfgToFile(t, Include{Input: inputInclude()}, filepath.Join(dir, "inc.toml"))

	err = app.Merge(NewIncludeDB(nil), &mockResolver{})

	require.ErrorIs(t, err, ErrIncludeIDNotFound)
	assert.Contains(t, err.Error(), "Tasks.t")
}
