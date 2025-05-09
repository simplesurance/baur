//go:build dbtest
// +build dbtest

package command

import (
	"encoding/csv"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v5/internal/command/flag"
	"github.com/simplesurance/baur/v5/internal/exec"
	"github.com/simplesurance/baur/v5/internal/testutils/dbtest"
	"github.com/simplesurance/baur/v5/internal/testutils/gittest"
	"github.com/simplesurance/baur/v5/internal/testutils/repotest"
	"github.com/simplesurance/baur/v5/pkg/baur"
)

func runInitDb(t *testing.T) {
	t.Helper()

	t.Log("creating database schema")
	initDb(initDbCmd, nil)
}

// baurCSVLsApps runs "baur ls apps --format=csv" and returns a slice where each
// element is a slice of csv fields per line
func baurCSVLsApps(t *testing.T) [][]string {
	t.Helper()

	stdoutBuf, _ := interceptCmdOutput(t)

	lsAppsCmd := newLsAppsCmd()
	lsAppsCmd.format.Val = flag.FormatCSV

	lsAppsCmd.Run(&lsAppsCmd.Command, nil)

	statusOut, err := csv.NewReader(stdoutBuf).ReadAll()
	require.NoError(t, err)

	return statusOut
}

// baurCSVLsInputs runs "baur ls inputs --format=csv" and returns the list of files as
// string slice. args is passed as argument to the "baur ls inputs" command.
func baurCSVLsInputs(t *testing.T, args ...string) []string {
	t.Helper()

	stdoutBuf, _ := interceptCmdOutput(t)

	lsInputsCmd := newLsInputsCmd()
	lsInputsCmd.format.Val = "csv"

	lsInputsCmd.Run(&lsInputsCmd.Command, args)

	return strings.Split(stdoutBuf.String(), "\n")
}

type csvStatus struct {
	taskID string
	status string
	commit string
}

// baurCSVStatusCmd runs the statusCmd, parses the CSV result and returns it.
// cmd.csv is set to true
func baurCSVStatusCmd(t *testing.T, cmd *statusCmd) []*csvStatus {
	t.Helper()

	stdoutBuf, _ := interceptCmdOutput(t)

	cmd.format.Val = "csv"
	cmd.quiet = true
	err := cmd.Execute()
	require.NoError(t, err)

	statusOut, err := csv.NewReader(stdoutBuf).ReadAll()
	require.NoError(t, err)

	result := make([]*csvStatus, 0, len(statusOut))

	for _, line := range statusOut {
		require.Len(t, line, 5)
		result = append(result, &csvStatus{
			taskID: line[0],
			status: line[3],
			commit: line[4],
		})
	}

	return result
}

// baurCSVStatus runs "baur status --format=csv" and returns the result.
func baurCSVStatus(t *testing.T, inputStr []string, lookupInputStr string) []*csvStatus {
	t.Helper()

	statusCmd := newStatusCmd()
	statusCmd.format.Val = "csv"
	statusCmd.quiet = true
	statusCmd.inputStr = inputStr
	statusCmd.lookupInputStr = lookupInputStr

	return baurCSVStatusCmd(t, statusCmd)
}

func assertStatusTasks(t *testing.T, r *repotest.Repo, statusOut []*csvStatus, expectedStatus baur.TaskStatus, commit string) {
	taskIDs := make([]string, 0, len(statusOut))
	for _, task := range statusOut {
		taskIDs = append(taskIDs, task.taskID)

		assert.Equal(t, expectedStatus.String(), task.status)

		if commit != "" {
			assert.Equal(t, commit, task.commit)
		}
	}

	assert.ElementsMatch(t, taskIDs, r.TaskIDs(), "baur status is missing some tasks")
}

// TestRunningPendingTasksChangesStatus creates a new baur repository with 2
// simple apps, one with inputs, one without, then runs:
// - "baur status", ensures all apps are listed and have status pending,
// - "baur run", ensures it was successful,
// - "baur status", ensures all apps are listed and have status run exist
// The test is running in 2 variants where the baur git repository is part of a
// git repo and where it is not.
func TestRunningPendingTasksChangesStatus(t *testing.T) {
	commit := ""

	initTest(t)

	r := repotest.CreateBaurRepository(t, repotest.WithNewDB())
	r.CreateSimpleApp(t)

	runInitDb(t)

	gittest.CommitFilesToGit(t, ".")

	res, err := exec.Command("git", "rev-parse", "HEAD").ExpectSuccess().RunCombinedOut(t.Context())
	require.NoError(t, err)

	commit = strings.TrimSpace(res.StrOutput())

	statusOut := baurCSVStatus(t, nil, "")
	assertStatusTasks(t, r, statusOut, baur.TaskStatusExecutionPending, "")

	runCmd := newRunCmd()
	runCmd.Run(&runCmd.Command, nil)

	statusOut = baurCSVStatus(t, nil, "")
	assertStatusTasks(t, r, statusOut, baur.TaskStatusRunExist, commit)
}

// TestRunningPendingTasksWithInputStringChangesStatus creates a new baur repository with a
// a app that has 2 tasks and ensures that baur run, baur ls inputs and baur
// status honors the --input-strings parameters of these commands.
func TestRunningPendingTasksWithInputStringChangesStatus(t *testing.T) {
	initTest(t)

	r := repotest.CreateBaurRepository(t, repotest.WithNewDB())
	r.CreateSimpleApp(t)

	runInitDb(t)

	gittest.CommitFilesToGit(t, ".")

	res, err := exec.Command("git", "rev-parse", "HEAD").ExpectSuccess().RunCombinedOut(t.Context())
	require.NoError(t, err)

	commit := strings.TrimSpace(res.StrOutput())

	// run 1, without input-strings
	runCmd := newRunCmd()
	runCmd.Run(&runCmd.Command, nil)

	statusOut := baurCSVStatus(t, nil, "")
	assertStatusTasks(t, r, statusOut, baur.TaskStatusRunExist, commit)

	// ensure status is Pending when input-strings are passed to "baur status"
	inputStr := []string{"feature-x", "branch-y"}
	statusOut = baurCSVStatus(t, inputStr, "")
	assertStatusTasks(t, r, statusOut, baur.TaskStatusExecutionPending, "")

	// run 2, with "feature-x", "feature-y" input strings
	runID := "3" // 3 instead of 2 becoes the app has 2 task, we build both
	runCmd = newRunCmd()
	runCmd.inputStr = inputStr
	runCmd.Run(&runCmd.Command, nil)

	statusOut = baurCSVStatus(t, inputStr, "")
	assertStatusTasks(t, r, statusOut, baur.TaskStatusRunExist, commit)

	out := baurCSVLsInputs(t, runID)
	require.Contains(t, out, "string:feature-x")
	require.Contains(t, out, "string:branch-y")

	// ensure status is Pending when only one of the two input-strings is
	// passed to "baur status"
	inputStr = []string{"feature-x"}
	statusOut = baurCSVStatus(t, inputStr, "")
	assertStatusTasks(t, r, statusOut, baur.TaskStatusExecutionPending, "")

	// run 3, with only "feature-x", input string
	runID = "5"
	runCmd = newRunCmd()
	runCmd.inputStr = inputStr
	runCmd.Run(&runCmd.Command, nil)

	statusOut = baurCSVStatus(t, inputStr, "")
	assertStatusTasks(t, r, statusOut, baur.TaskStatusRunExist, commit)

	out = baurCSVLsInputs(t, runID)
	assert.Contains(t, out, "string:feature-x")
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
	runCmd.inputStr = []string{featureX}
	runCmd.Run(&runCmd.Command, nil)

	statusOut := baurCSVStatus(t, []string{featureX}, "")
	assertStatusTasks(t, r, statusOut, baur.TaskStatusRunExist, "")

	featureY := "feature-y"

	statusOut = baurCSVStatus(t, []string{featureY}, "")
	assertStatusTasks(t, r, statusOut, baur.TaskStatusExecutionPending, "")

	statusOut = baurCSVStatus(t, []string{featureY}, featureX)
	assertStatusTasks(t, r, statusOut, baur.TaskStatusRunExist, "")
}

func TestAppWithoutTasks(t *testing.T) {
	initTest(t)

	r := repotest.CreateBaurRepository(t, repotest.WithNewDB())
	appCfg := r.CreateAppWithoutTasks(t)

	runInitDb(t)

	statusOut := baurCSVStatus(t, nil, "")

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

	gittest.CreateRepository(t, testdataDir)

	t.Chdir(filepath.Join(testdataDir, "var_in_include"))

	gittest.CommitFilesToGit(t, testdataDir)

	dbURL, err := dbtest.CreateDB(dbtest.UniqueDBName())
	require.NoError(t, err)

	t.Setenv(envVarPSQLURL, dbURL)

	runInitDb(t)

	runCmd := newRunCmd()
	runCmd.Run(&runCmd.Command, []string{"app1", "app2"})
}
