//go:build dbtest

package command

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v5/internal/testutils/fstest"
	"github.com/simplesurance/baur/v5/internal/testutils/gittest"
	"github.com/simplesurance/baur/v5/pkg/baur"
	"github.com/simplesurance/baur/v5/pkg/cfg"
)

func statusByTask(t *testing.T) map[string]string {
	t.Helper()

	result := map[string]string{}
	for _, s := range baurCSVStatus(t, nil, "") {
		result[s.taskID] = s.status
	}

	return result
}

func TestRunFailingTaskExitsWithErrorAndOtherTasksStillRun(t *testing.T) {
	r := newRepoWithTasks(t,
		shTask("a_fail", "echo failure-marker >&2; exit 3", nil),
		shTask("b_ok", "true", nil),
	)
	_, stderrBuf := interceptCmdOutput(t)

	runTasks(t, exitCodeError)

	assert.Contains(t, stderrBuf.String(), "svc.a_fail")
	assert.Contains(t, stderrBuf.String(), "failure-marker", "the output of a failed task is shown")

	assert.Len(t, storedTaskRunIDs(t, r), 1, "only the successful run is recorded")
	assert.Equal(t, map[string]string{
		"svc.a_fail": baur.TaskStatusExecutionPending.String(),
		"svc.b_ok":   baur.TaskStatusRunExist.String(),
	}, statusByTask(t))
}

func TestRunFailFastSkipsQueuedTasks(t *testing.T) {
	r := newRepoWithTasks(t,
		shTask("a_fail", "exit 1", nil),
		shTask("b_ok", "true", nil),
	)
	_, stderrBuf := interceptCmdOutput(t)

	runTasks(t, exitCodeError, "--fail-fast")

	assert.Contains(t, stderrBuf.String(), "skipping")
	assert.Empty(t, storedTaskRunIDs(t, r), "tasks queued after a failure must not run")
	assert.Equal(t, map[string]string{
		"svc.a_fail": baur.TaskStatusExecutionPending.String(),
		"svc.b_ok":   baur.TaskStatusExecutionPending.String(),
	}, statusByTask(t))
}

func TestRunForceRunsTasksWithExistingRun(t *testing.T) {
	r := newRepoWithTasks(t, shTask("a", "true", nil), shTask("b", "true", nil))
	stdoutBuf, _ := interceptCmdOutput(t)

	runTasks(t, 0)
	assert.Contains(t, stdoutBuf.String(), "Running 2/2 task(s) with status")
	assert.Len(t, storedTaskRunIDs(t, r), 2)

	stdoutBuf.Reset()
	runTasks(t, 0)
	assert.Contains(t, stdoutBuf.String(), "Running 0/2 task(s) with status")
	assert.Len(t, storedTaskRunIDs(t, r), 2, "tasks with an existing run must not run again")

	stdoutBuf.Reset()
	runTasks(t, 0, "--force")
	assert.Contains(t, stdoutBuf.String(), "Running 2/2 task(s) with status")
	assert.Len(t, storedTaskRunIDs(t, r), 4, "--force creates new runs")
}

func TestRunSkipUploadDoesNotUploadOrRecord(t *testing.T) {
	artifactDir := t.TempDir()
	out := []cfg.FileOutput{{
		Path:     "out.txt",
		FileCopy: []cfg.FileCopy{{Path: artifactDir}},
	}}

	r := newRepoWithTasks(t, shTask("build", "echo data > out.txt", out))
	stdoutBuf, _ := interceptCmdOutput(t)

	runTasks(t, 0, "--skip-upload")

	assert.Contains(t, stdoutBuf.String(), "--skip-upload was passed")
	assert.Empty(t, storedTaskRunIDs(t, r))
	assert.Equal(t, baur.TaskStatusExecutionPending.String(), statusByTask(t)["svc.build"])
	assert.NoFileExists(t, filepath.Join(artifactDir, "out.txt"))

	runTasks(t, 0)

	assert.Len(t, storedTaskRunIDs(t, r), 1)
	assert.Equal(t, baur.TaskStatusRunExist.String(), statusByTask(t)["svc.build"])
	assert.Equal(t, "data\n", string(fstest.ReadFile(t, filepath.Join(artifactDir, "out.txt"))))
}

func TestRunParallelRunsMustBeGreaterThanZero(t *testing.T) {
	r := newRepoWithTasks(t, shTask("a", "true", nil))
	_, stderrBuf := interceptCmdOutput(t)

	runTasks(t, exitCodeError, "-p", "0")

	assert.Contains(t, stderrBuf.String(), "--parallel-runs must be greater than 0")
	assert.Empty(t, storedTaskRunIDs(t, r))
}

func TestRunRerunsOnlyTasksWithChangedInputs(t *testing.T) {
	r := newRepoWithTasks(t,
		shTask("build", "true", nil, "in.txt"),
		shTask("check", "true", nil),
	)
	inFile := filepath.Join(r.Dir, testAppName, "in.txt")
	fstest.WriteToFile(t, []byte("1"), inFile)
	interceptCmdOutput(t)

	runTasks(t, 0)
	require.Len(t, storedTaskRunIDs(t, r), 2)
	assert.Equal(t, map[string]string{
		"svc.build": baur.TaskStatusRunExist.String(),
		"svc.check": baur.TaskStatusRunExist.String(),
	}, statusByTask(t))

	fstest.WriteToFile(t, []byte("2"), inFile)

	assert.Equal(t, map[string]string{
		"svc.build": baur.TaskStatusExecutionPending.String(),
		"svc.check": baur.TaskStatusRunExist.String(),
	}, statusByTask(t), "only the task using the changed file is pending")

	// statusByTask replaces the intercepted output streams
	stdoutBuf, _ := interceptCmdOutput(t)
	runTasks(t, 0)

	assert.Contains(t, stdoutBuf.String(), "Running 1/2 task(s) with status")
	assert.Len(t, storedTaskRunIDs(t, r), 3)

	// restoring the old content makes the first run valid again
	fstest.WriteToFile(t, []byte("1"), inFile)
	stdoutBuf, _ = interceptCmdOutput(t)
	runTasks(t, 0)
	assert.Contains(t, stdoutBuf.String(), "Running 0/2 task(s) with status")
	assert.Len(t, storedTaskRunIDs(t, r), 3)
}

func TestRunFailsWhenDeclaredOutputIsNotCreated(t *testing.T) {
	out := []cfg.FileOutput{{
		Path:     "missing.txt",
		FileCopy: []cfg.FileCopy{{Path: t.TempDir()}},
	}}

	r := newRepoWithTasks(t, shTask("build", "true", out))
	_, stderrBuf := interceptCmdOutput(t)

	runTasks(t, exitCodeError)

	assert.Contains(t, stderrBuf.String(), "was not created by the task run")
	assert.Empty(t, storedTaskRunIDs(t, r), "a run with missing outputs must not be recorded")
}

func TestRunWithUnknownTargetFails(t *testing.T) {
	r := newRepoWithTasks(t, shTask("a", "true", nil))

	runTasks(t, exitCodeError, "doesnotexist.build")

	assert.Empty(t, storedTaskRunIDs(t, r))
}

func TestRunRequireCleanGitWorktree(t *testing.T) {
	r := newRepoWithTasks(t, shTask("a", "true", nil))
	gittest.CommitFilesToGit(t, r.Dir)

	t.Run("clean worktree", func(t *testing.T) {
		runTasks(t, 0, "--require-clean-git-worktree")
		assert.Len(t, storedTaskRunIDs(t, r), 1)
	})

	t.Run("untracked file", func(t *testing.T) {
		const untracked = "untracked-file.txt"
		require.NoError(t, os.WriteFile(filepath.Join(r.Dir, untracked), []byte("x"), 0o600))
		_, stderrBuf := interceptCmdOutput(t)

		runTasks(t, exitCodeError, "--force", "--require-clean-git-worktree")

		assert.Contains(t, stderrBuf.String(), "--require-clean-git-worktree")
		assert.Contains(t, stderrBuf.String(), untracked)
		assert.Len(t, storedTaskRunIDs(t, r), 1, "no task must run when the worktree is not clean")
	})

	t.Run("untracked file is ignored without the flag", func(t *testing.T) {
		runTasks(t, 0, "--force")
		assert.Len(t, storedTaskRunIDs(t, r), 2)
	})
}
