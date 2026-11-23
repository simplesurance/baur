package command

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v5/pkg/baur"
	"github.com/simplesurance/baur/v5/pkg/cfg"
)

func TestInitRepo(t *testing.T) {
	initTest(t)

	t.Run("creates the config in the passed directory", func(t *testing.T) {
		stdoutBuf, _ := interceptCmdOutput(t)
		dir := t.TempDir()
		cfgPath := filepath.Join(dir, baur.RepositoryCfgFile)

		initRepo(initRepoCmd, []string{dir})

		got, err := cfg.RepositoryFromFile(cfgPath)
		require.NoError(t, err)
		assert.Equal(t, cfg.ExampleRepository().Database.PGSQLURL, got.Database.PGSQLURL)
		assert.Equal(t, cfg.Version, got.ConfigVersion)
		assert.Equal(t, []string{"."}, got.Discover.Dirs)

		assert.Contains(t, stdoutBuf.String(), "Repository configuration was written to "+cfgPath)
		assert.Contains(t, stdoutBuf.String(), "Next Steps")
		assert.Contains(t, stdoutBuf.String(), cmdInitDb)
		assert.Contains(t, stdoutBuf.String(), cmdInitApp)
	})

	t.Run("without argument the current directory is used", func(t *testing.T) {
		interceptCmdOutput(t)
		dir := t.TempDir()
		t.Chdir(dir)

		initRepo(initRepoCmd, nil)

		assert.FileExists(t, filepath.Join(dir, baur.RepositoryCfgFile))
	})

	t.Run("existing config is not overwritten", func(t *testing.T) {
		_, stderrBuf := interceptCmdOutput(t)
		dir := t.TempDir()
		cfgPath := filepath.Join(dir, baur.RepositoryCfgFile)
		require.NoError(t, os.WriteFile(cfgPath, []byte("keep me"), 0o600))

		expectExit(t, exitCodeError, func() { initRepo(initRepoCmd, []string{dir}) })

		assert.Contains(t, stderrBuf.String(), cfgPath+" already exist")

		content, err := os.ReadFile(cfgPath)
		require.NoError(t, err)
		assert.Equal(t, "keep me", string(content))
	})

	t.Run("directory that does not exist", func(t *testing.T) {
		_, stderrBuf := interceptCmdOutput(t)
		dir := filepath.Join(t.TempDir(), "missing")

		expectExit(t, exitCodeError, func() { initRepo(initRepoCmd, []string{dir}) })

		assert.NotEmpty(t, stderrBuf.String())
		assert.NoFileExists(t, filepath.Join(dir, baur.RepositoryCfgFile))
	})

	t.Run("at most 1 argument is accepted", func(t *testing.T) {
		require.NoError(t, initRepoCmd.Args(initRepoCmd, nil))
		require.NoError(t, initRepoCmd.Args(initRepoCmd, []string{"a"}))
		require.Error(t, initRepoCmd.Args(initRepoCmd, []string{"a", "b"}))
	})
}
