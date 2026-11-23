package cfg

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateIncludes(t *testing.T) {
	require.NoError(t, validateIncludes(nil))
	require.NoError(t, validateIncludes([]string{"inc.toml#id", "../shared/inc.toml#other id"}))

	t.Run("absolute path", func(t *testing.T) {
		err := validateIncludes([]string{filepath.FromSlash("/abs/inc.toml") + "#id"})

		require.ErrorContains(t, err, "absolute path")
	})

	for _, spec := range []string{"", "#", "a"} {
		t.Run("invalid spec "+spec, func(t *testing.T) {
			require.ErrorContains(t, validateIncludes([]string{spec}), "invalid include specifier")
		})
	}

	// the format is <RELATIVE-FILEPATH>#<INCLUDE-ID>, both parts are required
	for _, spec := range []string{"#id", "nohash", "path#", "a#b#c"} {
		t.Run("invalid spec "+spec, func(t *testing.T) {
			require.ErrorContains(t, validateIncludes([]string{spec}), "invalid include specifier")
		})
	}

	t.Run("one character path and id are valid", func(t *testing.T) {
		require.NoError(t, validateIncludes([]string{"a#b"}))
	})

	t.Run("the first invalid element is reported", func(t *testing.T) {
		err := validateIncludes([]string{"ok.toml#id", "#"})

		require.ErrorContains(t, err, "#")
	})
}

func TestValidateIncludeID(t *testing.T) {
	require.NoError(t, validateIncludeID("build_inputs"))
	require.NoError(t, validateIncludeID("with space"))

	require.ErrorContains(t, validateIncludeID(""), "is empty")
	require.ErrorContains(t, validateIncludeID("a#b"), "contains invalid character '#'")
	require.ErrorContains(t, validateIncludeID(" "), "only whitespaces")
	require.ErrorContains(t, validateIncludeID("\t \n"), "only whitespaces")
}

func TestValidateUniqIncludeIDs(t *testing.T) {
	t.Run("distinct ids", func(t *testing.T) {
		incl := Include{
			Input:  InputIncludes{{IncludeID: "i"}},
			Output: OutputIncludes{{IncludeID: "o"}},
			Task:   TaskIncludes{{IncludeID: "t"}},
		}

		require.NoError(t, incl.validateUniqIncludeIDs())
	})

	tests := []struct {
		name string
		incl Include
	}{
		{"two inputs", Include{Input: InputIncludes{{IncludeID: "x"}, {IncludeID: "x"}}}},
		{"two outputs", Include{Output: OutputIncludes{{IncludeID: "x"}, {IncludeID: "x"}}}},
		{"two tasks", Include{Task: TaskIncludes{{IncludeID: "x"}, {IncludeID: "x"}}}},
		{"input and output", Include{Input: InputIncludes{{IncludeID: "x"}}, Output: OutputIncludes{{IncludeID: "x"}}}},
		{"input and task", Include{Input: InputIncludes{{IncludeID: "x"}}, Task: TaskIncludes{{IncludeID: "x"}}}},
		{"output and task", Include{Output: OutputIncludes{{IncludeID: "x"}}, Task: TaskIncludes{{IncludeID: "x"}}}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.incl.validateUniqIncludeIDs()

			require.ErrorContains(t, err, `includeID "x"`)
		})
	}
}

func TestIncludeFromFile(t *testing.T) {
	dir := t.TempDir()

	t.Run("sets the file path in all sections", func(t *testing.T) {
		p := filepath.Join(dir, "inc.toml")
		require.NoError(t, os.WriteFile(p, []byte(`
[[Input]]
include_id = "in"
[[Input.Files]]
paths = ["a"]

[[Output]]
include_id = "out"
[[Output.File]]
path = "f"

[[Task]]
include_id = "task"
name = "build"
command = ["make"]
`), 0o600))

		incl, err := IncludeFromFile(p)

		require.NoError(t, err)
		assert.Equal(t, p, incl.filePath)
		assert.Equal(t, p, incl.Input[0].filepath)
		assert.Equal(t, p, incl.Output[0].filepath)
		assert.Contains(t, incl.Task[0].cfgFiles, p)
		assert.Equal(t, "build", incl.Task[0].Name)
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := IncludeFromFile(filepath.Join(dir, "missing.toml"))

		require.ErrorIs(t, err, os.ErrNotExist)
	})

	t.Run("invalid toml", func(t *testing.T) {
		p := filepath.Join(dir, "bad.toml")
		require.NoError(t, os.WriteFile(p, []byte("[[Input"), 0o600))

		_, err := IncludeFromFile(p)

		require.Error(t, err)
	})
}

func TestExampleIncludeWrittenAndReadIsValid(t *testing.T) {
	p := filepath.Join(t.TempDir(), "inc.toml")
	require.NoError(t, ExampleInclude().ToFile(p))

	got, err := IncludeFromFile(p)
	require.NoError(t, err)

	assert.Len(t, got.Input, 1)
	assert.Len(t, got.Output, 1)
	assert.Len(t, got.Task, 1)
}

func TestValidateUniqIncludeIDsReportsTheSectionOfTheDuplicate(t *testing.T) {
	tests := []struct {
		name     string
		incl     Include
		wantPath string
	}{
		{
			"input",
			Include{Input: InputIncludes{{IncludeID: "x"}, {IncludeID: "x"}}},
			"Input.include_id",
		},
		{
			"output duplicates an input",
			Include{Input: InputIncludes{{IncludeID: "x"}}, Output: OutputIncludes{{IncludeID: "x"}}},
			"Output.include_id",
		},
		{
			"task duplicates an output",
			Include{Output: OutputIncludes{{IncludeID: "x"}}, Task: TaskIncludes{{IncludeID: "x"}}},
			"Task.include_id",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.incl.validateUniqIncludeIDs()

			require.ErrorContains(t, err, tc.wantPath)
			require.ErrorContains(t, err, `"x"`)
		})
	}
}
