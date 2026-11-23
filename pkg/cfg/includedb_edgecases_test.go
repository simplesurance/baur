package cfg

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseIncludeSpec(t *testing.T) {
	db := NewIncludeDB(nil)

	t.Run("relative path is joined with the working dir", func(t *testing.T) {
		abs, id, err := db.parseIncludeSpec(&mockResolver{}, "/work", "sub/inc.toml#my id")

		require.NoError(t, err)
		assert.Equal(t, filepath.FromSlash("/work/sub/inc.toml"), abs)
		assert.Equal(t, "my id", id)
	})

	t.Run("absolute path is kept", func(t *testing.T) {
		abs, _, err := db.parseIncludeSpec(&mockResolver{}, "/work", filepath.FromSlash("/abs/inc.toml")+"#id")

		require.NoError(t, err)
		assert.Equal(t, filepath.FromSlash("/abs/inc.toml"), abs)
	})

	t.Run("nil resolver is allowed", func(t *testing.T) {
		abs, _, err := db.parseIncludeSpec(nil, "/work", "inc.toml#id")

		require.NoError(t, err)
		assert.Equal(t, filepath.FromSlash("/work/inc.toml"), abs)
	})

	t.Run("resolver is applied to the path only", func(t *testing.T) {
		r := funcResolver(func(s string) (string, error) { return s + ".resolved", nil })

		abs, id, err := db.parseIncludeSpec(r, "/abs", filepath.FromSlash("/x/inc")+"#id")

		require.NoError(t, err)
		assert.Equal(t, filepath.FromSlash("/x/inc.resolved"), abs)
		assert.Equal(t, "id", id)
	})

	t.Run("resolver error", func(t *testing.T) {
		_, _, err := db.parseIncludeSpec(failingResolverOn("inc.toml"), "/work", "inc.toml#id")

		require.ErrorContains(t, err, "resolving variables failed")
	})

	for _, spec := range []string{"inc.toml", "a#b#c", ""} {
		t.Run("invalid spec "+spec, func(t *testing.T) {
			_, _, err := db.parseIncludeSpec(&mockResolver{}, "/work", spec)

			require.ErrorContains(t, err, "exactly one '#'")
		})
	}
}

func TestNewIncludeDBLogsAndAcceptsNilLogger(t *testing.T) {
	dir := t.TempDir()
	cfgToFile(t, Include{Input: inputInclude()}, filepath.Join(dir, "inc.toml"))

	var logs []string
	db := NewIncludeDB(func(format string, _ ...any) { logs = append(logs, format) })
	_, err := db.loadInputInclude(&mockResolver{}, dir, "inc.toml#inputs")
	require.NoError(t, err)
	assert.NotEmpty(t, logs)

	_, err = NewIncludeDB(nil).loadInputInclude(&mockResolver{}, dir, "inc.toml#inputs")
	require.NoError(t, err)
}

func TestIncludeFileIsLoadedOnlyOnce(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "inc.toml")
	cfgToFile(t, Include{Input: inputInclude(), Output: outputInclude(), Task: taskInclude()}, p)
	db := NewIncludeDB(t.Logf)

	first, err := db.loadInputInclude(&mockResolver{}, dir, "inc.toml#inputs")
	require.NoError(t, err)

	require.NoError(t, os.Remove(p))

	second, err := db.loadInputInclude(&mockResolver{}, dir, "inc.toml#inputs")
	require.NoError(t, err, "second load must come from the db")
	assert.Same(t, first, second)

	_, err = db.loadOutputInclude(&mockResolver{}, dir, "inc.toml#outputs")
	require.NoError(t, err, "all includes of the file are available after it was loaded once")

	_, err = db.loadTaskInclude(&mockResolver{}, dir, "inc.toml#build_task")
	require.NoError(t, err)
}

func TestLoadIncludeWrongIncludeKindIsNotFound(t *testing.T) {
	dir := t.TempDir()
	cfgToFile(t, Include{Input: inputInclude(), Output: outputInclude(), Task: taskInclude()}, filepath.Join(dir, "inc.toml"))
	db := NewIncludeDB(nil)

	_, err := db.loadInputInclude(&mockResolver{}, dir, "inc.toml#outputs")
	require.ErrorIs(t, err, ErrIncludeIDNotFound)

	_, err = db.loadOutputInclude(&mockResolver{}, dir, "inc.toml#inputs")
	require.ErrorIs(t, err, ErrIncludeIDNotFound)

	_, err = db.loadTaskInclude(&mockResolver{}, dir, "inc.toml#inputs")
	require.ErrorIs(t, err, ErrIncludeIDNotFound)
}

func TestLoadIncludeFromFileWithoutThatKindIsNotFound(t *testing.T) {
	dir := t.TempDir()
	cfgToFile(t, Include{Input: inputInclude()}, filepath.Join(dir, "only-inputs.toml"))
	cfgToFile(t, Include{Output: outputInclude()}, filepath.Join(dir, "only-outputs.toml"))

	t.Run("output include requested from a file with inputs only", func(t *testing.T) {
		db := NewIncludeDB(nil)
		_, err := db.loadInputInclude(&mockResolver{}, dir, "only-inputs.toml#inputs")
		require.NoError(t, err)

		_, err = db.loadOutputInclude(&mockResolver{}, dir, "only-inputs.toml#inputs")
		require.ErrorIs(t, err, ErrIncludeIDNotFound)
	})

	t.Run("input include requested from a file with outputs only", func(t *testing.T) {
		db := NewIncludeDB(nil)
		_, err := db.loadOutputInclude(&mockResolver{}, dir, "only-outputs.toml#outputs")
		require.NoError(t, err)

		_, err = db.loadInputInclude(&mockResolver{}, dir, "only-outputs.toml#outputs")
		require.ErrorIs(t, err, ErrIncludeIDNotFound)
	})
}

func TestLoadIncludeErrors(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600))
	}

	write("badtoml.toml", "[[Input")
	write("dupids.toml", `
[[Input]]
include_id = "x"
[[Input.Files]]
paths = ["a"]
[[Input]]
include_id = "x"
`)
	write("badinput.toml", `
[[Input]]
include_id = "x"
[[Input.Files]]
paths = [""]
`)
	write("badoutput.toml", `
[[Output]]
include_id = "x"
`)
	write("badtask.toml", `
[[Task]]
include_id = "x"
name = "build"
`)
	write("unknowninclude.toml", `
[[Task]]
include_id = "x"
name = "build"
command = ["make"]
includes = ["unknown.toml#nothing"]
`)

	tests := []struct {
		file    string
		id      string
		wantErr string
	}{
		{"badtoml.toml", "x", "badtoml.toml"},
		{"dupids.toml", "x", "includeIDs must be unique"},
		{"badinput.toml", "x", "Input"},
		{"badoutput.toml", "x", "no output is defined"},
		{"badtask.toml", "x", "command"},
		{"unknowninclude.toml", "x", "merge failed"},
	}

	for _, tc := range tests {
		t.Run(tc.file, func(t *testing.T) {
			_, err := NewIncludeDB(nil).loadTaskInclude(&mockResolver{}, dir, tc.file+"#"+tc.id)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
			assert.NotErrorIs(t, err, ErrIncludeIDNotFound)
		})
	}

	t.Run("not existing file keeps os.ErrNotExist", func(t *testing.T) {
		_, err := NewIncludeDB(nil).loadTaskInclude(&mockResolver{}, dir, "nope.toml#x")

		require.ErrorIs(t, err, os.ErrNotExist)
	})

	t.Run("invalid include specifier mentions the include", func(t *testing.T) {
		_, err := NewIncludeDB(nil).loadTaskInclude(&mockResolver{}, dir, "nohash")

		require.ErrorContains(t, err, "exactly one '#'")
	})

	t.Run("resolver failure", func(t *testing.T) {
		_, err := NewIncludeDB(nil).loadInputInclude(failingResolverOn("a.toml"), dir, "a.toml#x")

		require.ErrorContains(t, err, "resolving variables failed")
	})
}

func TestAddIncludeRejectsDuplicates(t *testing.T) {
	db := NewIncludeDB(nil)

	require.NoError(t, db.addInputInclude("/f", &InputInclude{IncludeID: "x"}))
	require.ErrorContains(t, db.addInputInclude("/f", &InputInclude{IncludeID: "x"}), "already exist")
	require.NoError(t, db.addInputInclude("/other", &InputInclude{IncludeID: "x"}), "same id in another file is fine")

	require.NoError(t, db.addOutputInclude("/f", &OutputInclude{IncludeID: "x"}))
	require.ErrorContains(t, db.addOutputInclude("/f", &OutputInclude{IncludeID: "x"}), "already exist")

	require.NoError(t, db.addTaskInclude("/f", &TaskInclude{IncludeID: "x"}))
	require.ErrorContains(t, db.addTaskInclude("/f", &TaskInclude{IncludeID: "x"}), "already exist")
}

func TestErrIncludeIDNotFoundIsDetectableViaErrorsIs(t *testing.T) {
	dir := t.TempDir()
	cfgToFile(t, Include{Input: inputInclude()}, filepath.Join(dir, "inc.toml"))

	_, err := NewIncludeDB(nil).loadInputInclude(&mockResolver{}, dir, "inc.toml#missing")

	require.ErrorIs(t, err, ErrIncludeIDNotFound)
}
