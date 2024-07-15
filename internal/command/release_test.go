//go:build dbtest

package command

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/simplesurance/baur/v5/internal/prettyprint"
	"github.com/simplesurance/baur/v5/internal/testutils/fstest"
	"github.com/simplesurance/baur/v5/internal/testutils/ostest"
	"github.com/simplesurance/baur/v5/internal/testutils/repotest"
	"github.com/simplesurance/baur/v5/pkg/baur"

	"github.com/stretchr/testify/assert"
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

	t.Run("CreateAndShowAllTasks", func(t *testing.T) {
		initTest(t)
		releaseCmd := newReleaseCreateCmd()

		releaseCmd.SetArgs([]string{"all"})
		require.NotPanics(t, func() { require.NoError(t, releaseCmd.Execute()) })

		showStdout, _ := interceptCmdOutput(t)
		showCmd := newReleaseShowCmd()
		showCmd.SetArgs([]string{"all"})
		require.NotPanics(t, func() { require.NoError(t, showCmd.Execute()) })

		// a map is used instead of baur.Release, to prevent that tests
		// succeed if the fields in the Release struct are renamed and
		// become downwards incompatible
		release := map[string]any{}
		err := json.Unmarshal(showStdout.Bytes(), &release)
		require.NoError(t, err, showStdout.String())
		t.Log(prettyprint.AsString(release))

		assert.Equal(t, "all", release["ReleaseName"])
		assert.NotContains(t, release, "Metadata")

		require.Contains(t, release, "Applications")
		assert.Len(t, release["Applications"], 1)
		apps := release["Applications"].(map[string]any)

		assert.Contains(t, apps, "simpleApp")
		assert.Contains(t, apps["simpleApp"], "TaskRuns")
		app := apps["simpleApp"].(map[string]any)
		taskRuns := app["TaskRuns"].(map[string]any)
		assert.Len(t, taskRuns, 2)
		require.Contains(t, taskRuns, "build")
		require.Contains(t, taskRuns, "check")

		checkTask := taskRuns["check"].(map[string]any)
		assert.Empty(t, checkTask["Outputs"])

		buildTask := taskRuns["build"].(map[string]any)
		require.Contains(t, buildTask, "Outputs")
		outputs := buildTask["Outputs"].(map[string]any)
		assert.Len(t, outputs, 1)
		require.Contains(t, outputs, "1")
		output := outputs["1"].(map[string]any)
		require.Contains(t, output, "Uploads")
		uploads := output["Uploads"].([]any)
		require.Len(t, uploads, 1)
		upload := uploads[0].(map[string]any)
		require.Contains(t, upload, "UploadMethod")
		assert.Equal(t, "filecopy", upload["UploadMethod"])
		require.Contains(t, upload, "URI")
		assert.NotEmpty(t, upload["URI"])
	})

	t.Run("CreateFailsWithAlreadyExists", func(t *testing.T) {
		initTest(t)

		releaseCmd := newReleaseCreateCmd()
		releaseCmd.SetArgs([]string{"all"})

		execCheck(t, releaseCmd, exitCodeAlreadyExist)
	})

	t.Run("CreateWithCommaSeparatedIncludes", func(t *testing.T) {
		initTest(t)
		releaseCmd := newReleaseCreateCmd()

		releaseCmd.SetArgs([]string{
			"--include", "*.build", "--include", "*.check,simpleApp.build", t.Name(),
		})
		require.NotPanics(t, func() { require.NoError(t, releaseCmd.Execute()) })
	})

	t.Run("CreateAndShowWithMetadataAndMultipleIncludes", func(t *testing.T) {
		initTest(t)
		releaseCmd := newReleaseCreateCmd()
		metadataSrcFilepath := r.AppCfgs[0].FilePath()

		releaseCmd.SetArgs([]string{
			"--include", "*.build", "--include", "*.check", "buildCheck",
			"-m", metadataSrcFilepath,
		})
		require.NotPanics(t, func() { require.NoError(t, releaseCmd.Execute()) })

		showCmd := newReleaseShowCmd()
		metadataFile := filepath.Join(t.TempDir(), "metadata")
		showCmd.SetArgs([]string{"-m", metadataFile, "buildCheck"})
		require.NotPanics(t, func() { require.NoError(t, showCmd.Execute()) })

		metadata := fstest.ReadFile(t, metadataFile)
		metadataSrc := fstest.ReadFile(t, metadataSrcFilepath)
		assert.Equal(t, metadataSrc, metadata)

		showCmd = newReleaseShowCmd()
		showStdout, _ := interceptCmdOutput(t)
		showCmd.SetArgs([]string{"buildCheck"})
		require.NotPanics(t, func() { require.NoError(t, showCmd.Execute()) })

		var release baur.Release
		err := json.Unmarshal(showStdout.Bytes(), &release)
		require.NoError(t, err, showStdout.String())
		require.Equal(t, string(metadataSrc), string(release.Metadata))
	})

	t.Run("ExistsWithBaurCfg", func(t *testing.T) {
		initTest(t)

		releaseExistsCmd := newReleaseExistsCmd()
		releaseExistsCmd.SetArgs([]string{"buildCheck"})
		require.NotPanics(t, func() { require.NoError(t, releaseExistsCmd.Execute()) })
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
