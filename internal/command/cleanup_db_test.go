//go:build dbtest

package command

import (
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v5/internal/command/flag"
	"github.com/simplesurance/baur/v5/internal/testutils/repotest"
)

const datePast = "2000.01.01-00:00"

func dateFuture() string {
	return time.Now().Add(24 * time.Hour).Format(flag.DateTimeFormat)
}

// outputCount returns the number printed for label in the output of "baur cleanup db".
func outputCount(t *testing.T, out, label string) string {
	t.Helper()

	m := regexp.MustCompile(label + `\s+(\d+)`).FindStringSubmatch(out)
	require.Lenf(t, m, 2, "label %q not found in output:\n%s", label, out)

	return m[1]
}

func cleanupDb(t *testing.T, expectedExitCode int, args ...string) string {
	t.Helper()

	stdoutBuf, _ := interceptCmdOutput(t)

	cmd := newCleanupDbCmd()
	cmd.SetArgs(args)
	execCheck(t, cmd, expectedExitCode)

	return stdoutBuf.String()
}

// newRepoWithReleaseAndExtraRuns creates a repo with 1 app with 2 tasks, runs
// all tasks, creates the release "rel1" for these runs and runs the tasks a
// second time with a different input string, so that 2 additional task runs
// exist that are not part of a release.
func newRepoWithReleaseAndExtraRuns(t *testing.T) *repotest.Repo {
	t.Helper()

	r := newRepoWithTasks(t,
		shTask("a", "true", nil),
		shTask("b", "true", nil),
	)

	runTasks(t, 0)
	require.Len(t, storedTaskRunIDs(t, r), 2)

	releaseCmd := newReleaseCreateCmd()
	releaseCmd.SetArgs([]string{"rel1"})
	execCheck(t, releaseCmd, 0)

	runTasks(t, 0, "--input-str=other")
	require.Len(t, storedTaskRunIDs(t, r), 4)

	return r
}

func TestCleanupDbTaskRuns(t *testing.T) {
	r := newRepoWithReleaseAndExtraRuns(t)

	t.Run("records newer than --before are kept", func(t *testing.T) {
		out := cleanupDb(t, 0, "--force", "--before="+datePast, "taskruns")

		assert.Contains(t, out, "deleting task runs older than")
		assert.Contains(t, out, "tasks runs referenced by releases are kept")
		assert.Equal(t, "0", outputCount(t, out, "Task Runs:"))
		assert.Len(t, storedTaskRunIDs(t, r), 4)
	})

	t.Run("pretend reports but does not delete", func(t *testing.T) {
		out := cleanupDb(t, 0, "--force", "--pretend", "--before="+dateFuture(), "taskruns")

		assert.Contains(t, out, "pretending to delete task runs")
		assert.Equal(t, "2", outputCount(t, out, "Task Runs:"), "only runs not referenced by a release are deletable")
		assert.Len(t, storedTaskRunIDs(t, r), 4)
	})

	t.Run("deletes only task runs that are not part of a release", func(t *testing.T) {
		before := storedTaskRunIDs(t, r)

		out := cleanupDb(t, 0, "--force", "--before="+dateFuture(), "taskruns")

		assert.Contains(t, out, "successful")
		assert.Equal(t, "2", outputCount(t, out, "Task Runs:"))
		// only the string input "other" is exclusive to the deleted runs,
		// it is stored once and shared by both. Inputs of released runs are kept.
		assert.Equal(t, "1", outputCount(t, out, "Inputs:"))
		assert.Equal(t, "0", outputCount(t, out, "Apps:"), "app is still referenced by released runs")
		assert.Equal(t, "0", outputCount(t, out, "Tasks:"), "tasks are still referenced by released runs")

		after := storedTaskRunIDs(t, r)
		assert.Len(t, after, 2)
		assert.Subset(t, before, after)
		assert.True(t, releaseExists(t, r, "rel1"))
	})

	t.Run("after the release is deleted, the remaining runs and dangling records are removed", func(t *testing.T) {
		out := cleanupDb(t, 0, "--force", "--before="+dateFuture(), "releases")
		assert.Equal(t, "1", outputCount(t, out, "Releases:"))
		assert.False(t, releaseExists(t, r, "rel1"))
		assert.Len(t, storedTaskRunIDs(t, r), 2, "deleting a release must not delete task runs")

		out = cleanupDb(t, 0, "--force", "--before="+dateFuture(), "taskruns")
		assert.Equal(t, "2", outputCount(t, out, "Task Runs:"))
		assert.Equal(t, "2", outputCount(t, out, "Tasks:"))
		assert.Equal(t, "1", outputCount(t, out, "Apps:"))
		assert.Empty(t, storedTaskRunIDs(t, r))
	})
}

func TestCleanupDbReleases(t *testing.T) {
	r := newRepoWithReleaseAndExtraRuns(t)

	t.Run("release newer than --before is kept", func(t *testing.T) {
		out := cleanupDb(t, 0, "--force", "--before="+datePast, "releases")

		assert.Equal(t, "0", outputCount(t, out, "Releases:"))
		assert.True(t, releaseExists(t, r, "rel1"))
	})

	t.Run("pretend does not delete", func(t *testing.T) {
		out := cleanupDb(t, 0, "--force", "-p", "--before="+dateFuture(), "releases")

		assert.Contains(t, out, "pretending to delete releases")
		assert.Equal(t, "1", outputCount(t, out, "Releases:"))
		assert.True(t, releaseExists(t, r, "rel1"))
	})

	t.Run("without --force the command waits before deleting", func(t *testing.T) {
		start := time.Now()
		out := cleanupDb(t, 0, "--pretend", "--before="+dateFuture(), "releases")

		assert.GreaterOrEqual(t, time.Since(start), cleanupDbGracetime)
		assert.Contains(t, out, "starting in")
		assert.Contains(t, out, "starting deleting...")
	})

	t.Run("deletes the release", func(t *testing.T) {
		out := cleanupDb(t, 0, "--force", "--before="+dateFuture(), "releases")

		assert.Equal(t, "1", outputCount(t, out, "Releases:"))
		assert.False(t, releaseExists(t, r, "rel1"))
	})
}

func TestCleanupDbInvalidInvocation(t *testing.T) {
	newRepoWithReleaseAndExtraRuns(t)

	t.Run("--before is required", func(t *testing.T) {
		initTest(t)

		cmd := newCleanupDbCmd()
		cmd.SetArgs([]string{"--force", "releases"})

		err := cmd.Execute()
		require.ErrorContains(t, err, `required flag(s) "before" not set`)
	})

	t.Run("unparsable --before", func(t *testing.T) {
		initTest(t)

		cmd := newCleanupDbCmd()
		cmd.SetArgs([]string{"--before=yesterday", "releases"})

		require.Error(t, cmd.Execute())
	})

	t.Run("exactly 1 target argument is required", func(t *testing.T) {
		initTest(t)

		cmd := newCleanupDbCmd()
		cmd.SetArgs([]string{"--before=" + datePast})
		require.Error(t, cmd.Execute())

		cmd = newCleanupDbCmd()
		cmd.SetArgs([]string{"--before=" + datePast, "releases", "taskruns"})
		require.Error(t, cmd.Execute())
	})

	t.Run("unknown target exits with an error and deletes nothing", func(t *testing.T) {
		cleanupDb(t, exitCodeError, "--force", "--before="+dateFuture(), "everything")
	})
}

func TestCleanupDbWithoutRepositoryUsesEnvVar(t *testing.T) {
	r := newRepoWithReleaseAndExtraRuns(t)

	t.Chdir(t.TempDir())
	t.Setenv(envVarPSQLURL, r.Cfg.Database.PGSQLURL)

	out := cleanupDb(t, 0, "--force", "--before="+dateFuture(), "releases")

	assert.Equal(t, "1", outputCount(t, out, "Releases:"))
	assert.False(t, releaseExists(t, r, "rel1"))
}

func TestCleanupDbFailsWithoutDatabaseConfiguration(t *testing.T) {
	initTest(t)

	t.Chdir(t.TempDir())
	t.Setenv(envVarPSQLURL, "")
	_, stderrBuf := interceptCmdOutput(t)

	cmd := newCleanupDbCmd()
	cmd.SetArgs([]string{"--force", "--before=" + datePast, "releases"})
	execCheck(t, cmd, exitCodeError)

	assert.Contains(t, stderrBuf.String(), "baur repository not found")
}
