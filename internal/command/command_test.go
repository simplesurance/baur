// +build dbtest

package command

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v1"
	"github.com/simplesurance/baur/v1/internal/exec"
	"github.com/simplesurance/baur/v1/internal/testutils/dbtest"
	"github.com/simplesurance/baur/v1/internal/testutils/gittest"
	"github.com/simplesurance/baur/v1/internal/testutils/repotest"
)

var testdataDir string

func init() {
	wd, err := os.Getwd()
	if err != nil {
		panic(wd)
	}

	testdataDir = filepath.Join(wd, "testdata")
}

func runInitDb(t *testing.T) {
	t.Helper()

	t.Log("creating database schema")
	initDb(initDbCmd, nil)
}

// baurCSVLsApps runs "baur ls apps --csv" and returns a slice where each
// element is a slice of csv fields per line
func baurCSVLsApps(t *testing.T) [][]string {
	t.Helper()

	stdoutBuf, _ := interceptCmdOutput(t)

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
func baurCSVStatus(t *testing.T, inputStr, lookupInputStr string) []*csvStatus {
	t.Helper()

	stdoutBuf, _ := interceptCmdOutput(t)

	statusCmd := newStatusCmd()
	statusCmd.csv = true
	statusCmd.inputStr = inputStr
	statusCmd.lookupInputStr = lookupInputStr

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

func assertStatusTasks(t *testing.T, r *repotest.Repo, statusOut []*csvStatus, expectedStatus baur.TaskStatus, commit string) {
	var taskIds []string

	for _, task := range statusOut {
		taskIds = append(taskIds, task.taskID)

		assert.Equal(t, expectedStatus.String(), task.status)
		assert.Equal(t, commit, task.commit)
	}

	assert.ElementsMatch(t, taskIds, r.TaskIDs(), "baur status is missing some tasks")
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
			commit := ""

			initTest(t)

			r := repotest.CreateBaurRepository(t, repotest.WithNewDB())
			r.CreateSimpleApp(t)

			runInitDb(t)

			if tc.withGitRepository {
				gittest.CreateRepository(t, ".")

				gittest.CommitFilesToGit(t, ".")

				res, err := exec.Command("git", "rev-parse", "HEAD").ExpectSuccess().Run()
				assert.NoError(t, err)

				commit = strings.TrimSpace(res.StrOutput())
			}

			statusOut := baurCSVStatus(t, "", "")
			assertStatusTasks(t, r, statusOut, baur.TaskStatusExecutionPending, "")

			runCmd := newRunCmd()
			runCmd.Command.Run(&runCmd.Command, nil)

			statusOut = baurCSVStatus(t, "", "")
			assertStatusTasks(t, r, statusOut, baur.TaskStatusRunExist, commit)
		})
	}
}

// TestRunningPendingTasksWithInputStringChangesStatus creates a new baur repository with a
// simple app then runs:
// - "baur run without an input string"
// - "baur status without an input string", ensures all apps are listed and have status run exist
// - "baur status with an input string", ensures all apps are listed and have status pending,
// - "baur run with an input string"
// - "baur status with an input string", ensures all apps are listed and have status run exist
func TestRunningPendingTasksWithInputStringChangesStatus(t *testing.T) {
	initTest(t)

	r := repotest.CreateBaurRepository(t, repotest.WithNewDB())
	r.CreateSimpleApp(t)

	runInitDb(t)

	var commit string

	gittest.CreateRepository(t, ".")

	gittest.CommitFilesToGit(t, ".")

	res, err := exec.Command("git", "rev-parse", "HEAD").ExpectSuccess().Run()
	assert.NoError(t, err)

	commit = strings.TrimSpace(res.StrOutput())

	runCmd := newRunCmd()
	runCmd.Command.Run(&runCmd.Command, nil)

	statusOut := baurCSVStatus(t, "", "")
	assertStatusTasks(t, r, statusOut, baur.TaskStatusRunExist, commit)

	inputStr := "feature-x"

	statusOut = baurCSVStatus(t, inputStr, "")
	assertStatusTasks(t, r, statusOut, baur.TaskStatusExecutionPending, "")

	runCmd = newRunCmd()
	runCmd.inputStr = inputStr
	runCmd.Command.Run(&runCmd.Command, nil)

	statusOut = baurCSVStatus(t, inputStr, "")
	assertStatusTasks(t, r, statusOut, baur.TaskStatusRunExist, commit)
}

// TestLookupInputStringReturnsRunExistsStatusWhenInputStringRunExists creates a new baur repository with a
// simple app then runs:
// - "baur run with an input string of feature-x"
// - "baur status with an input string of feature-x", ensures all apps are listed and have status run exist
// - "baur status with an input string of feature-y", ensures all apps are listed and have status pending,
// - "baur status with an input string of feature-y and lookup input string of feature-x", ensures all apps are listed and have status run exist
func TestLookupInputStringReturnsRunExistsStatusWhenInputStringRunExists(t *testing.T) {
	initTest(t)

	r := repotest.CreateBaurRepository(t, repotest.WithNewDB())
	r.CreateSimpleApp(t)

	runInitDb(t)

	featureX := "feature-x"

	runCmd := newRunCmd()
	runCmd.inputStr = featureX
	runCmd.Command.Run(&runCmd.Command, nil)

	statusOut := baurCSVStatus(t, featureX, "")
	assertStatusTasks(t, r, statusOut, baur.TaskStatusRunExist, "")

	featureY := "feature-y"

	statusOut = baurCSVStatus(t, featureY, "")
	assertStatusTasks(t, r, statusOut, baur.TaskStatusExecutionPending, "")

	statusOut = baurCSVStatus(t, featureY, featureX)
	assertStatusTasks(t, r, statusOut, baur.TaskStatusRunExist, "")
}

func TestAppWithoutTasks(t *testing.T) {
	initTest(t)

	r := repotest.CreateBaurRepository(t, repotest.WithNewDB())
	appCfg := r.CreateAppWithoutTasks(t)

	runInitDb(t)

	statusOut := baurCSVStatus(t, "", "")

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

func TestVarInInclude(t *testing.T) {
	initTest(t)

	err := os.Chdir(filepath.Join(testdataDir, "var_in_include"))
	require.NoError(t, err)

	dbURL, err := dbtest.CreateDB(dbtest.UniqueDBName())
	require.NoError(t, err)

	oldEnvVal := os.Getenv(envVarPSQLURL)
	t.Cleanup(func() {
		os.Setenv(envVarPSQLURL, oldEnvVal)
	})

	err = os.Setenv(envVarPSQLURL, dbURL)
	require.NoError(t, err)

	runInitDb(t)

	runCmd := newRunCmd()
	runCmd.Command.Run(&runCmd.Command, []string{"app1", "app2"})
}
