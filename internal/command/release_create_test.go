//go:build dbtest

package command

import (
	"testing"

	"github.com/simplesurance/baur/v3/internal/testutils/repotest"

	"github.com/stretchr/testify/require"
)

func TestCreateRelease(t *testing.T) {
	initTest(t)

	r := repotest.CreateBaurRepository(t, repotest.WithNewDB())
	r.CreateSimpleApp(t)
	runInitDb(t)

	t.Run("failsWhenTaskRunsArePending", func(t *testing.T) {
		initTest(t)
		releaseCmd := newReleaseCreateCmd()
		releaseCmd.SetArgs([]string{"all"})

		execCheck(t, releaseCmd, exitCodeTaskRunIsPending)
	})

	runCmd := newRunCmd()
	require.NotPanics(t, func() { require.NoError(t, runCmd.Execute()) })

	// TODO: extend the testcases to verify the data of the created
	// release, when "release show" was implemented

	t.Run("allTasks", func(t *testing.T) {
		initTest(t)
		releaseCmd := newReleaseCreateCmd()

		releaseCmd.SetArgs([]string{"all"})
		require.NotPanics(t, func() { require.NoError(t, releaseCmd.Execute()) })
	})

	t.Run("releaseAlreadyExistsErr", func(t *testing.T) {
		initTest(t)

		releaseCmd := newReleaseCreateCmd()
		releaseCmd.SetArgs([]string{"all"})

		execCheck(t, releaseCmd, exitCodeAlreadyExist)
	})

	t.Run("metadataAndMultipleIncludes", func(t *testing.T) {
		initTest(t)
		releaseCmd := newReleaseCreateCmd()

		releaseCmd.SetArgs([]string{
			"--include", "*.build", "--include", "*.check", "buildCheck",
			"-m", r.AppCfgs[0].FilePath(),
		})
		require.NotPanics(t, func() { require.NoError(t, releaseCmd.Execute()) })
	})

}
