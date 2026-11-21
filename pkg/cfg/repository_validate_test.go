package cfg

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepositoryValidate(t *testing.T) {
	valid := ExampleRepository

	tests := []struct {
		name    string
		modify  func(*Repository)
		wantErr string
	}{
		{"example is valid", func(*Repository) {}, ""},
		{"unset version", func(r *Repository) { r.ConfigVersion = 0 }, "config_version: can not be unset or 0"},
		{"older version", func(r *Repository) { r.ConfigVersion = Version - 1 }, "incompatible configuration files"},
		{"newer version", func(r *Repository) { r.ConfigVersion = Version + 1 }, "incompatible configuration files"},
		{"no discover dirs", func(r *Repository) { r.Discover.Dirs = nil }, "Discover.application_dirs: can not be empty"},
		{"search depth 0 is allowed", func(r *Repository) { r.Discover.SearchDepth = 0 }, ""},
		{"search depth 10 is allowed", func(r *Repository) { r.Discover.SearchDepth = 10 }, ""},
		{"search depth 11", func(r *Repository) { r.Discover.SearchDepth = 11 }, "Discover.search_depth"},
		{"negative search depth", func(r *Repository) { r.Discover.SearchDepth = -1 }, "Discover.search_depth"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := valid()
			tc.modify(r)

			err := r.Validate()

			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

func TestRepositoryFromFileErrors(t *testing.T) {
	dir := t.TempDir()

	_, err := RepositoryFromFile(filepath.Join(dir, "missing.toml"))
	require.ErrorIs(t, err, os.ErrNotExist)

	bad := filepath.Join(dir, "bad.toml")
	require.NoError(t, os.WriteFile(bad, []byte("this is = = not toml"), 0o600))
	_, err = RepositoryFromFile(bad)
	require.Error(t, err)
}

func TestRepositoryFromFileParsesAllFields(t *testing.T) {
	p := filepath.Join(t.TempDir(), ".baur.toml")
	require.NoError(t, os.WriteFile(p, []byte(`
config_version = 7

[Database]
postgresql_url = "postgres://x@y/z"

[Discover]
application_dirs = ["a", "b/c"]
search_depth = 3
`), 0o600))

	got, err := RepositoryFromFile(p)

	require.NoError(t, err)
	assert.Equal(t, 7, got.ConfigVersion)
	assert.Equal(t, "postgres://x@y/z", got.Database.PGSQLURL)
	assert.Equal(t, []string{"a", "b/c"}, got.Discover.Dirs)
	assert.Equal(t, 3, got.Discover.SearchDepth)
	assert.Equal(t, p, got.FilePath())
}

func TestSearchDepthErrorDescribesTheAllowedRange(t *testing.T) {
	r := ExampleRepository()
	r.Discover.SearchDepth = maxSearchDepth + 1

	err := r.Discover.validate()

	require.ErrorContains(t, err, "[0, 10]", "0 is a valid value, the range is closed on both ends")
}
