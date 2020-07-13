package command

import (
	"bytes"
	"encoding/csv"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur"
	"github.com/simplesurance/baur/exec"
	"github.com/simplesurance/baur/internal/command/term"
	"github.com/simplesurance/baur/testutils/gittest"
	"github.com/simplesurance/baur/testutils/repotest"
)

func initTest(t *testing.T) {
	t.Helper()

	exitFunc = func(code int) {
		t.Fatalf("baur command exited with code %d", code)
	}

	exec.DefaultDebugfFn = t.Logf
}

// baurCSVStatus runs "baur status --csv" and returns a slice where each
// element is a slice of csv fields per line
func baurCSVStatus(t *testing.T) [][]string {
	var stdoutBuf bytes.Buffer

	t.Helper()

	stdout = term.NewStream(&stdoutBuf)

	statusCmd := newStatusCmd()
	statusCmd.csv = true

	statusCmd.Command.Run(&statusCmd.Command, nil)

	statusOut, err := csv.NewReader(&stdoutBuf).ReadAll()
	require.NoError(t, err)

	return statusOut
}

// baurCSVLsApps runs "baur ls apps --csv" and returns a slice where each
// element is a slice of csv fields per line
func baurCSVLsApps(t *testing.T) [][]string {
	var stdoutBuf bytes.Buffer

	t.Helper()

	stdout = term.NewStream(&stdoutBuf)

	lsAppsCmd := newLsAppsCmd()
	lsAppsCmd.csv = true

	lsAppsCmd.Command.Run(&lsAppsCmd.Command, nil)

	statusOut, err := csv.NewReader(&stdoutBuf).ReadAll()
	require.NoError(t, err)

	return statusOut
}

func runInitDb(t *testing.T) {
	t.Helper()

	t.Log("creating database schema")
	initDb(initDbCmd, nil)
}

// TestRunningPendingTasksChangesStatus creates a new baur repository with 2
// simple apps, one with inputs, one without, then runs:
// - "baur status", ensures all apps are listed and have status pending,
// - "baur run", ensures it was successful,
// - "baur status", ensures all apps are listed and have status run exist
// The test is running in 2 variants where the baur git repository is part of a
// git repo and where it is not.
func TestRunningPendingTasksChangesStatus(t *testing.T) {
	testcases := []struct {
		testname          string
		withGitRepository bool
	}{
		{
			testname:          "withoutGit",
			withGitRepository: false,
		},
		{
			testname:          "withGit",
			withGitRepository: true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.testname, func(t *testing.T) {
			var headCommit string

			initTest(t)

			r := repotest.CreateBaurRepository(t, repotest.WithNewDB())
			r.CreateSimpleApp(t)

			err := os.Chdir(r.Dir)
			require.NoError(t, err)

			runInitDb(t)

			if tc.withGitRepository {
				_, err := exec.Command("git", "init", ".").ExpectSuccess().Run()
				assert.NoError(t, err)

				gittest.CommitFilesToGit(t, ".")

				res, err := exec.Command("git", "rev-parse", "HEAD").ExpectSuccess().Run()
				assert.NoError(t, err)

				headCommit = strings.TrimSpace(res.StrOutput())
			}

			statusOut := baurCSVStatus(t)

			var taskIds []string
			for _, line := range statusOut {
				require.Equal(t, 5, len(line))
				taskID := line[0]
				status := line[3]
				commit := line[4]

				taskIds = append(taskIds, taskID)

				assert.Equal(t, status, baur.TaskStatusExecutionPending.String(), "status is not %s: %q", baur.TaskStatusExecutionPending, line)

				assert.Equal(t, "", commit)
			}
			assert.ElementsMatch(t, taskIds, r.TaskIDs(), "baur status is missing some tasks")

			runCmd := newRunCmd()
			runCmd.Command.Run(&runCmd.Command, nil)

			statusOut = baurCSVStatus(t)
			taskIds = nil
			for _, line := range statusOut {
				require.Equal(t, 5, len(line))
				taskID := line[0]
				status := line[3]
				commit := line[4]

				taskIds = append(taskIds, taskID)

				assert.Equal(t, status, baur.TaskStatusRunExist.String(), "status is not %s: %q", baur.TaskStatusRunExist, line)

				if tc.withGitRepository {
					assert.Equal(t, headCommit, commit)
				}
			}
			assert.ElementsMatch(t, taskIds, r.TaskIDs(), "baur status is missing some tasks")
		})
	}
}

func TestAppWithoutTasks(t *testing.T) {
	initTest(t)

	r := repotest.CreateBaurRepository(t, repotest.WithNewDB())
	appCfg := r.CreateAppWithoutTasks(t)

	runInitDb(t)

	statusOut := baurCSVStatus(t)

	assert.Empty(t, statusOut, "expected empty baur status output, got: %q", statusOut)

	lsAppsOut := baurCSVLsApps(t)

	var found bool
	for _, line := range lsAppsOut {
		require.Len(t, line, 2)
		name := line[0]

		if name == appCfg.Name {
			found = true
		}
	}

	assert.True(t, found, "baur ls apps did not list %q: %q", appCfg.Name, lsAppsOut)
}
