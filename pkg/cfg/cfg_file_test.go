package cfg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToFile(t *testing.T) {
	t.Run("writes parseable TOML", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), ".baur.toml")

		require.NoError(t, ExampleRepository().ToFile(p))

		got, err := RepositoryFromFile(p)
		require.NoError(t, err)
		assert.Equal(t, Version, got.ConfigVersion)
		assert.Equal(t, p, got.FilePath())
		require.NoError(t, got.Validate())
	})

	t.Run("existing file is not overwritten by default", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), ".baur.toml")
		require.NoError(t, os.WriteFile(p, []byte("keep"), 0o600))

		err := ExampleRepository().ToFile(p)

		require.ErrorIs(t, err, os.ErrExist)
		content, err := os.ReadFile(p)
		require.NoError(t, err)
		assert.Equal(t, "keep", string(content))
	})

	t.Run("overwrite option replaces the content", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), ".baur.toml")
		require.NoError(t, os.WriteFile(p, []byte(strings.Repeat("old", 1000)), 0o600))

		require.NoError(t, ExampleRepository().ToFile(p, ToFileOptOverwrite()))

		got, err := RepositoryFromFile(p)
		require.NoError(t, err, "file must be truncated, no remains of the old content")
		assert.Equal(t, Version, got.ConfigVersion)
	})

	t.Run("commented option prefixes every line", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), ".baur.toml")

		require.NoError(t, ExampleRepository().ToFile(p, ToFileOptCommented()))

		content, err := os.ReadFile(p)
		require.NoError(t, err)
		lines := strings.Split(strings.TrimSuffix(string(content), "\n"), "\n")
		require.NotEmpty(t, lines)
		for _, l := range lines {
			assert.True(t, strings.HasPrefix(l, "# "), "line %q is not commented", l)
		}
	})

	t.Run("not writable path returns an error", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "missing-dir", ".baur.toml")

		require.Error(t, ExampleRepository().ToFile(p))
	})
}
