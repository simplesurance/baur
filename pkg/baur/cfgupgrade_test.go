package baur

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v5/pkg/cfg"
)

func writeRepoCfg(t *testing.T, dir string, version int) string {
	t.Helper()

	r := cfg.ExampleRepository()
	r.ConfigVersion = version

	p := filepath.Join(dir, RepositoryCfgFile)
	require.NoError(t, r.ToFile(p))

	return p
}

func TestCfgUpgraderIsUpToDate(t *testing.T) {
	t.Run("current version", func(t *testing.T) {
		dir := t.TempDir()
		writeRepoCfg(t, dir, cfg.Version)

		ok, err := NewCfgUpgrader(dir).IsUpToDate()
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("older version", func(t *testing.T) {
		dir := t.TempDir()
		writeRepoCfg(t, dir, 5)

		ok, err := NewCfgUpgrader(dir).IsUpToDate()
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("missing config is an error", func(t *testing.T) {
		_, err := NewCfgUpgrader(t.TempDir()).IsUpToDate()
		require.Error(t, err)
	})
}

func TestCfgUpgraderUpgrade(t *testing.T) {
	for _, from := range []int{5, 6} {
		t.Run(fmt.Sprintf("upgrades from v%d and keeps a backup", from), func(t *testing.T) {
			dir := t.TempDir()
			p := writeRepoCfg(t, dir, from)

			before, err := os.ReadFile(p)
			require.NoError(t, err)

			require.NoError(t, NewCfgUpgrader(dir).Upgrade())

			upgraded, err := cfg.RepositoryFromFile(p)
			require.NoError(t, err)
			assert.Equal(t, cfg.Version, upgraded.ConfigVersion)

			bak, err := os.ReadFile(p + ".bak")
			require.NoError(t, err)
			assert.Equal(t, string(before), string(bak), "backup must contain the original file")

			ok, err := NewCfgUpgrader(dir).IsUpToDate()
			require.NoError(t, err)
			assert.True(t, ok)
		})
	}

	for _, v := range []int{1, 4, cfg.Version, cfg.Version + 1} {
		t.Run(fmt.Sprintf("rejects v%d and leaves the file untouched", v), func(t *testing.T) {
			dir := t.TempDir()
			p := writeRepoCfg(t, dir, v)
			before, err := os.ReadFile(p)
			require.NoError(t, err)

			err = NewCfgUpgrader(dir).Upgrade()

			require.ErrorContains(t, err, "only supported from version 5 and 6")
			after, err := os.ReadFile(p)
			require.NoError(t, err)
			assert.Equal(t, string(before), string(after))
			assert.NoFileExists(t, p+".bak")
		})
	}

	t.Run("missing config is an error", func(t *testing.T) {
		require.Error(t, NewCfgUpgrader(t.TempDir()).Upgrade())
	})
}
