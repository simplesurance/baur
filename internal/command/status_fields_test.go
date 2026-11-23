//go:build dbtest

package command

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v5/internal/exec"
	"github.com/simplesurance/baur/v5/internal/testutils/gittest"
	"github.com/simplesurance/baur/v5/pkg/baur"
)

const allStatusFields = "--fields=app-name,task-id,path,status,run-id,git-commit"

func statusOutput(t *testing.T, expectedExitCode int, args ...string) string {
	t.Helper()

	stdoutBuf, _ := interceptCmdOutput(t)

	cmd := newStatusCmd()
	cmd.SetArgs(args)
	execCheck(t, cmd, expectedExitCode)

	return stdoutBuf.String()
}

func statusCSV(t *testing.T, args ...string) [][]string {
	t.Helper()

	recs, err := csv.NewReader(strings.NewReader(statusOutput(t, 0, append([]string{"--format=csv"}, args...)...))).ReadAll()
	require.NoError(t, err)

	return recs
}

// newRepoWithOneTaskRun creates a repo with the tasks svc.a, svc.b and runs
// only svc.a.
func newRepoWithOneTaskRun(t *testing.T) (repoDir, commit string) {
	t.Helper()

	r := newRepoWithTasks(t, shTask("a", "true", nil), shTask("b", "true", nil))
	gittest.CommitFilesToGit(t, r.Dir)

	res, err := exec.Command("git", "rev-parse", "HEAD").ExpectSuccess().RunCombinedOut(t.Context())
	require.NoError(t, err)

	runTasks(t, 0, "svc.a")

	return r.Dir, strings.TrimSpace(res.StrOutput())
}

func TestStatusFields(t *testing.T) {
	repoDir, commit := newRepoWithOneTaskRun(t)
	exist := baur.TaskStatusRunExist.String()
	pending := baur.TaskStatusExecutionPending.String()

	t.Run("all fields with headers", func(t *testing.T) {
		recs := statusCSV(t, allStatusFields)

		assert.Equal(t, [][]string{
			{statusAppNameHeader, statusTaskIDHeader, statusPathHeader, statusStatusHeader, statusRunIDHeader, statusGitCommitHeader},
			{"svc", "svc.a", "svc", exist, "1", commit},
			{"svc", "svc.b", "svc", pending, "", ""},
		}, recs)
	})

	t.Run("quiet omits the header", func(t *testing.T) {
		recs := statusCSV(t, "-q", "--fields=task-id,status")

		assert.Equal(t, [][]string{
			{"svc.a", exist},
			{"svc.b", pending},
		}, recs)
	})

	t.Run("fields that do not need the database", func(t *testing.T) {
		t.Setenv(envVarPSQLURL, "postgres://invalid.invalid:1/none?connect_timeout=1")

		recs := statusCSV(t, "-q", "--fields=app-name,task-id,path")

		assert.Equal(t, [][]string{
			{"svc", "svc.a", "svc"},
			{"svc", "svc.b", "svc"},
		}, recs)
	})

	t.Run("--abs-path", func(t *testing.T) {
		recs := statusCSV(t, "-q", "--abs-path", "--fields=task-id,path")

		assert.Equal(t, [][]string{
			{"svc.a", filepath.Join(repoDir, "svc")},
			{"svc.b", filepath.Join(repoDir, "svc")},
		}, recs)
	})

	t.Run("--status filters tasks", func(t *testing.T) {
		assert.Equal(t, [][]string{{"svc.a"}}, statusCSV(t, "-q", "--fields=task-id", "-s", "exist"))
		assert.Equal(t, [][]string{{"svc.b"}}, statusCSV(t, "-q", "--fields=task-id", "-s", "pending"))
	})

	t.Run("target argument selects tasks", func(t *testing.T) {
		assert.Equal(t, [][]string{{"svc.b"}}, statusCSV(t, "-q", "--fields=task-id", "svc.b"))
	})

	t.Run("invalid --status value", func(t *testing.T) {
		initTest(t)

		cmd := newStatusCmd()
		cmd.SetArgs([]string{"-s", "bogus"})

		require.ErrorContains(t, cmd.Execute(), "status must be")
	})

	t.Run("json format", func(t *testing.T) {
		var got []map[string]any
		out := statusOutput(t, 0, "--format=json", allStatusFields)
		require.NoError(t, json.Unmarshal([]byte(out), &got), out)

		assert.Equal(t, []map[string]any{
			{"AppName": "svc", "TaskID": "svc.a", "Path": "svc", "Status": exist, "RunID": "1", "GitCommit": commit},
			{"AppName": "svc", "TaskID": "svc.b", "Path": "svc", "Status": pending, "RunID": nil, "GitCommit": nil},
		}, got)
	})

	t.Run("plain format with header", func(t *testing.T) {
		out := statusOutput(t, 0, "--fields=task-id,status")

		lines := strings.Split(strings.TrimSpace(out), "\n")
		require.GreaterOrEqual(t, len(lines), 3, out)
		assert.Contains(t, lines[0], statusTaskIDHeader)
		assert.Contains(t, lines[0], statusStatusHeader)
		assert.Contains(t, out, "svc.a")
		assert.Contains(t, out, "svc.b")
	})

	t.Run("--require-clean-git-worktree with untracked file", func(t *testing.T) {
		_, stderrBuf := interceptCmdOutput(t)
		require.NoError(t, os.WriteFile(filepath.Join(repoDir, "untracked.txt"), []byte("x"), 0o600))

		cmd := newStatusCmd()
		cmd.SetArgs([]string{"-c"})
		execCheck(t, cmd, exitCodeError)

		assert.Contains(t, stderrBuf.String(), "untracked.txt")
	})
}
