package baur

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newShellTask(dir, script string) *Task {
	return &Task{
		ID:             "app.build",
		AppName:        "app",
		Name:           "build",
		Directory:      dir,
		RepositoryRoot: dir,
		Command:        []string{"sh", "-c", script},
	}
}

func newQuietTaskRunner(skipAfterError bool) *TaskRunner {
	tr := NewTaskRunner(nil, skipAfterError)
	tr.LogFn = func(string, ...any) {}

	return tr
}

func TestTaskRunnerRun(t *testing.T) {
	t.Run("successful command", func(t *testing.T) {
		res, err := newQuietTaskRunner(false).Run(newShellTask(t.TempDir(), "exit 0"))
		require.NoError(t, err)

		assert.Equal(t, 0, res.ExitCode)
		assert.False(t, res.StartTime.IsZero())
		assert.False(t, res.StopTime.Before(res.StartTime))
	})

	t.Run("failing command returns its exit code without an error", func(t *testing.T) {
		res, err := newQuietTaskRunner(false).Run(newShellTask(t.TempDir(), "exit 3"))
		require.NoError(t, err)

		assert.Equal(t, 3, res.ExitCode)
	})

	t.Run("command runs in the task directory", func(t *testing.T) {
		dir := t.TempDir()

		_, err := newQuietTaskRunner(false).Run(newShellTask(dir, "pwd > where.txt"))
		require.NoError(t, err)

		got, err := os.ReadFile(filepath.Join(dir, "where.txt"))
		require.NoError(t, err)
		wantDir, err := filepath.EvalSymlinks(dir)
		require.NoError(t, err)
		assert.Equal(t, wantDir, strings.TrimSpace(string(got)))
	})

	t.Run("command inherits the environment of the process", func(t *testing.T) {
		t.Setenv("BAUR_TEST_INHERITED", "yes")
		dir := t.TempDir()

		_, err := newQuietTaskRunner(false).Run(newShellTask(dir, `printf %s "$BAUR_TEST_INHERITED" > env.txt`))
		require.NoError(t, err)

		got, err := os.ReadFile(filepath.Join(dir, "env.txt"))
		require.NoError(t, err)
		assert.Equal(t, "yes", string(got))
	})

	t.Run("command output is passed to LogFn", func(t *testing.T) {
		tr := NewTaskRunner(nil, false)
		var logged strings.Builder
		tr.LogFn = func(format string, v ...any) {
			logged.WriteString(strings.TrimSpace(fmt.Sprintf(format, v...)))
		}

		_, err := tr.Run(newShellTask(t.TempDir(), "echo hello-from-task"))
		require.NoError(t, err)

		assert.Contains(t, logged.String(), "hello-from-task")
	})

	t.Run("nonexistent command is an error", func(t *testing.T) {
		task := newShellTask(t.TempDir(), "")
		task.Command = []string{"/nonexistent/binary-xyz"}

		_, err := newQuietTaskRunner(false).Run(task)
		require.Error(t, err)
	})
}

func TestTaskRunnerGitUntrackedFilesFn(t *testing.T) {
	t.Run("clean worktree runs the task", func(t *testing.T) {
		tr := newQuietTaskRunner(false)
		var gotDir string
		tr.GitUntrackedFilesFn = func(dir string) ([]string, error) {
			gotDir = dir
			return nil, nil
		}
		task := newShellTask(t.TempDir(), "exit 0")

		_, err := tr.Run(task)
		require.NoError(t, err)
		assert.Equal(t, task.RepositoryRoot, gotDir)
	})

	t.Run("untracked files prevent the run and are reported", func(t *testing.T) {
		tr := newQuietTaskRunner(false)
		tr.GitUntrackedFilesFn = func(string) ([]string, error) { return []string{"a", "b"}, nil }
		dir := t.TempDir()

		_, err := tr.Run(newShellTask(dir, "touch ran.txt"))

		var eu *ErrUntrackedGitFilesExist
		require.ErrorAs(t, err, &eu)
		assert.Equal(t, []string{"a", "b"}, eu.UntrackedFiles)
		assert.NotEmpty(t, eu.Error())
		assert.NoFileExists(t, filepath.Join(dir, "ran.txt"), "the command must not run")
	})

	t.Run("error from the git function is returned and the task does not run", func(t *testing.T) {
		boom := errors.New("git failed")
		tr := newQuietTaskRunner(false)
		tr.GitUntrackedFilesFn = func(string) ([]string, error) { return nil, boom }
		dir := t.TempDir()

		_, err := tr.Run(newShellTask(dir, "touch ran.txt"))

		require.ErrorIs(t, err, boom)
		assert.NoFileExists(t, filepath.Join(dir, "ran.txt"))
	})
}

func TestTaskRunnerSkipRuns(t *testing.T) {
	t.Run("enabled with skipAfterError skips the run", func(t *testing.T) {
		tr := newQuietTaskRunner(true)
		assert.False(t, tr.SkipRunsIsEnabled())

		tr.SkipRuns(true)
		assert.True(t, tr.SkipRunsIsEnabled())

		dir := t.TempDir()
		_, err := tr.Run(newShellTask(dir, "touch ran.txt"))

		require.ErrorIs(t, err, ErrTaskRunSkipped)
		assert.NoFileExists(t, filepath.Join(dir, "ran.txt"))
	})

	t.Run("can be disabled again", func(t *testing.T) {
		tr := newQuietTaskRunner(true)
		tr.SkipRuns(true)
		tr.SkipRuns(false)

		assert.False(t, tr.SkipRunsIsEnabled())
		_, err := tr.Run(newShellTask(t.TempDir(), "exit 0"))
		require.NoError(t, err)
	})

	t.Run("without skipAfterError the flag has no effect", func(t *testing.T) {
		tr := newQuietTaskRunner(false)
		tr.SkipRuns(true)

		_, err := tr.Run(newShellTask(t.TempDir(), "exit 0"))
		require.NoError(t, err)
	})
}
