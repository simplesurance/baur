//go:build dbtest

package command

import (
	"testing"

	"github.com/simplesurance/baur/v3/internal/testutils/ostest"
	"github.com/simplesurance/baur/v3/internal/testutils/repotest"

	"github.com/stretchr/testify/require"
)

func TestRelease(t *testing.T) {
	initTest(t)

	r := repotest.CreateBaurRepository(t, repotest.WithNewDB())
	r.CreateSimpleApp(t)
	runInitDb(t)

	t.Run("ExistsNonExistingRelease", func(t *testing.T) {
		initTest(t)

		releaseExistsCmd := newReleaseExistsCmd()
		releaseExistsCmd.SetArgs([]string{"abc"})
		execCheck(t, releaseExistsCmd, exitCodeNotExist)
	})

	t.Run("CreatefailsWhenTaskRunsArePending", func(t *testing.T) {
		initTest(t)
		releaseCmd := newReleaseCreateCmd()
		releaseCmd.SetArgs([]string{"all"})

		execCheck(t, releaseCmd, exitCodeTaskRunIsPending)
	})

	runCmd := newRunCmd()
	require.NotPanics(t, func() { require.NoError(t, runCmd.Execute()) })

	// TODO: extend the testcases to verify the data of the created
	// release, when "release show" was implemented

	t.Run("CreateAllTasksAndExistsSucceeds", func(t *testing.T) {
		initTest(t)
		releaseCmd := newReleaseCreateCmd()

		releaseCmd.SetArgs([]string{"all"})
		require.NotPanics(t, func() { require.NoError(t, releaseCmd.Execute()) })

		releaseExistsCmd := newReleaseExistsCmd()
		releaseExistsCmd.SetArgs([]string{"all"})
		require.NotPanics(t, func() { require.NoError(t, releaseExistsCmd.Execute()) })
	})

	t.Run("CreateFailsWithAlreadyExists", func(t *testing.T) {
		initTest(t)

		releaseCmd := newReleaseCreateCmd()
		releaseCmd.SetArgs([]string{"all"})

		execCheck(t, releaseCmd, exitCodeAlreadyExist)
	})

	t.Run("CreateWithMetadataAndMultipleIncludes", func(t *testing.T) {
		initTest(t)
		releaseCmd := newReleaseCreateCmd()

		releaseCmd.SetArgs([]string{
			"--include", "*.build", "--include", "*.check", "buildCheck",
			"-m", r.AppCfgs[0].FilePath(),
		})
		require.NotPanics(t, func() { require.NoError(t, releaseCmd.Execute()) })
	})

	t.Run("ExistsWithPsqlURIviaEnv", func(t *testing.T) {
		initTest(t)

		ostest.Chdir(t, t.TempDir())

		t.Setenv(envVarPSQLURL, r.Cfg.Database.PGSQLURL)

		releaseExistsCmd := newReleaseExistsCmd()
		releaseExistsCmd.SetArgs([]string{"buildCheck"})
		require.NotPanics(t, func() { require.NoError(t, releaseExistsCmd.Execute()) })
	})
}
