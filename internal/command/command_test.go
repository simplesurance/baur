// +build dbtest

package command

import (
	"encoding/csv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v1"
	"github.com/simplesurance/baur/v1/exec"
	"github.com/simplesurance/baur/v1/internal/testutils/gittest"
	"github.com/simplesurance/baur/v1/internal/testutils/repotest"
)

func runInitDb(t *testing.T) {
	t.Helper()

	t.Log("creating database schema")
	initDb(initDbCmd, nil)
}

// baurCSVLsApps runs "baur ls apps --csv" and returns a slice where each
// element is a slice of csv fields per line
func baurCSVLsApps(t *testing.T) [][]string {
	t.Helper()

	stdoutBuf, _ := interceptCmdOutput()

	lsAppsCmd := newLsAppsCmd()
	lsAppsCmd.csv = true

	lsAppsCmd.Command.Run(&lsAppsCmd.Command, nil)

	statusOut, err := csv.NewReader(stdoutBuf).ReadAll()
	require.NoError(t, err)

	return statusOut
}

type csvStatus struct {
	taskID string
	status string
	commit string
}

// baurCSVStatus runs "baur status --csv" and returns the result.
func baurCSVStatus(t *testing.T) []*csvStatus {
	t.Helper()

	stdoutBuf, _ := interceptCmdOutput()

	statusCmd := newStatusCmd()
	statusCmd.csv = true

	statusCmd.Command.Run(&statusCmd.Command, nil)

	statusOut, err := csv.NewReader(stdoutBuf).ReadAll()
	require.NoError(t, err)

	result := make([]*csvStatus, 0, len(statusOut))

	for _, line := range statusOut {
		require.Equal(t, 5, len(line))
		result = append(result, &csvStatus{
			taskID: line[0],
			status: line[3],
			commit: line[4],
		})
	}

	return result
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
			for _, task := range statusOut {
				taskIds = append(taskIds, task.taskID)

				assert.Equal(t, baur.TaskStatusExecutionPending.String(), task.status)
				assert.Equal(t, "", task.commit)
			}
			assert.ElementsMatch(t, taskIds, r.TaskIDs(), "baur status is missing some tasks")

			runCmd := newRunCmd()
			runCmd.Command.Run(&runCmd.Command, nil)

			statusOut = baurCSVStatus(t)
			taskIds = nil
			for _, task := range statusOut {
				taskIds = append(taskIds, task.taskID)

				assert.Equal(t, baur.TaskStatusRunExist.String(), task.status)

				if tc.withGitRepository {
					assert.Equal(t, headCommit, task.commit)
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
