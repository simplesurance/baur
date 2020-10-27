// +build dbtest

package command

import (
	"encoding/csv"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v1/internal/testutils/repotest"
)

var appOneName = "app_one"
var appTwoName = "app_two"
var buildTaskName = "build"
var testTaskName = "test"
var appOneWithBuildTask = fmt.Sprintf("%s.%s", appOneName, buildTaskName)
var appOneWithTestTask = fmt.Sprintf("%s.%s", appOneName, testTaskName)
var appTwoWithBuildTask = fmt.Sprintf("%s.%s", appTwoName, buildTaskName)

func doInitDb(t *testing.T) {
	t.Helper()

	t.Log("creating database schema")
	initDb(initDbCmd, nil)
}

func assertExitCode(t *testing.T, expected int) {
	exitFunc = func(code int) {
		assert.Equal(t, expected, code)
		t.SkipNow()
	}
}

func Test2ArgsRequired(t *testing.T) {
	testcases := []struct {
		testname string
		args     []string
	}{
		{
			testname: "withNoArgs",
			args:     []string{},
		},
		{
			testname: "with3Args",
			args:     []string{"1", "2", "3"},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.testname, func(t *testing.T) {
			redirectOutputToLogger(t)
			r := repotest.CreateBaurRepository(t)
			r.CreateAppWithNoOutputs(t, appOneName)

			diffInputsCmd := newDiffInputsCmd()
			diffInputsCmd.SetArgs(tc.args)
			err := diffInputsCmd.Execute()

			assert.EqualError(t, err, fmt.Sprintf("accepts 2 args, received %d", len(tc.args)))
		})
	}
}

func TestWildCardsNotAllowed(t *testing.T) {
	testcases := []struct {
		testname string
		appTask  string
	}{
		{
			testname: "withAppOnlyWildCard",
			appTask:  "*.task",
		},
		{
			testname: "withAppContainingWildCard",
			appTask:  "app*.task",
		},
		{
			testname: "withTaskOnlyWildCard",
			appTask:  "app.*",
		},
		{
			testname: "withTaskContainingWildCard",
			appTask:  "app.task*",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.testname, func(t *testing.T) {
			redirectOutputToLogger(t)
			r := repotest.CreateBaurRepository(t)
			r.CreateAppWithNoOutputs(t, appOneName)

			diffInputsCmd := newDiffInputsCmd()
			diffInputsCmd.SetArgs([]string{tc.appTask, "app.task"})
			err := diffInputsCmd.Execute()

			assert.EqualError(t, err, fmt.Sprintf("invalid argument: \"%s\"", tc.appTask))
		})
	}
}

func TestAppAndTaskRequired(t *testing.T) {
	testcases := []struct {
		testname string
		appTask  string
	}{
		{
			testname: "withoutTaskAndSeparator",
			appTask:  "app",
		},
		{
			testname: "withoutTask",
			appTask:  "app.",
		},
		{
			testname: "withoutApp",
			appTask:  ".task",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.testname, func(t *testing.T) {
			redirectOutputToLogger(t)
			r := repotest.CreateBaurRepository(t)
			r.CreateAppWithNoOutputs(t, appOneName)

			diffInputsCmd := newDiffInputsCmd()
			diffInputsCmd.SetArgs([]string{tc.appTask, "app.task"})
			err := diffInputsCmd.Execute()

			assert.EqualError(t, err, fmt.Sprintf("invalid argument: \"%s\"", tc.appTask))
		})
	}
}

func TestUnknownAppOrTaskReturnsExitCode1(t *testing.T) {
	testcases := []struct {
		testname string
		appTask  string
	}{
		{
			testname: "withUnknownApp",
			appTask:  fmt.Sprintf("%s.%s", "unknown", buildTaskName),
		},
		{
			testname: "withUnknownTask",
			appTask:  fmt.Sprintf("%s.%s", appOneName, "unknown"),
		},
		{
			testname: "withUnknownAppAndTask",
			appTask:  fmt.Sprintf("%s.%s", "unknown", "unknown"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.testname, func(t *testing.T) {
			redirectOutputToLogger(t)
			r := repotest.CreateBaurRepository(t)
			r.CreateAppWithNoOutputs(t, appOneName)

			assertExitCode(t, 1)

			diffInputsCmd := newDiffInputsCmd()
			diffInputsCmd.SetArgs([]string{tc.appTask, appOneWithBuildTask})
			executeWithoutError(t, diffInputsCmd)
		})
	}
}

func TestCurrentInputsAgainstSameTaskCurrentInputsReturnsExitCode1(t *testing.T) {
	redirectOutputToLogger(t)
	r := repotest.CreateBaurRepository(t)
	r.CreateAppWithNoOutputs(t, appOneName)

	diffInputsCmd := newDiffInputsCmd()
	diffInputsCmd.SetArgs([]string{appOneWithBuildTask, appOneWithBuildTask})
	err := diffInputsCmd.Execute()
	assert.EqualError(t, err, fmt.Sprintf("%s and %s refer to the same task-run", appOneWithBuildTask, appOneWithBuildTask))
}

func TestNonExistentRunReturnsExitCode1(t *testing.T) {
	testcases := []struct {
		testname string
		run      string
	}{
		{
			testname: "withCaret",
			run:      fmt.Sprintf("%s^^^^", appOneWithBuildTask),
		},
		{
			testname: "withRunID",
			run:      "99",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.testname, func(t *testing.T) {
			redirectOutputToLogger(t)
			r := repotest.CreateBaurRepository(t, repotest.WithNewDB())
			r.CreateAppWithNoOutputs(t, appOneName)

			doInitDb(t)

			runCmd := newRunCmd()
			runCmd.run(&runCmd.Command, []string{appOneWithBuildTask})

			assertExitCode(t, 1)

			diffInputsCmd := newDiffInputsCmd()
			diffInputsCmd.SetArgs([]string{appOneWithBuildTask, tc.run})
			executeWithoutError(t, diffInputsCmd)
		})
	}
}

func TestCurrentInputsAgainstPreviousRunThatHasSameInputsReturnsExitCode0(t *testing.T) {
	testcases := []struct {
		testname    string
		previousRun string
	}{
		{
			testname:    "withCaret",
			previousRun: fmt.Sprintf("%s^", appOneWithBuildTask),
		},
		{
			testname:    "withRunID",
			previousRun: "1",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.testname, func(t *testing.T) {
			redirectOutputToLogger(t)
			r := repotest.CreateBaurRepository(t, repotest.WithNewDB())
			r.CreateAppWithNoOutputs(t, appOneName)

			doInitDb(t)

			runCmd := newRunCmd()
			runCmd.run(&runCmd.Command, []string{appOneWithBuildTask})

			assertExitCode(t, 0)

			diffInputsCmd := newDiffInputsCmd()
			diffInputsCmd.SetArgs([]string{appOneWithBuildTask, tc.previousRun})
			executeWithoutError(t, diffInputsCmd)
		})
	}
}

func TestPreviousRunAgainstAnotherPreviousRunThatHasSameInputsReturnsExitCode0(t *testing.T) {
	testcases := []struct {
		testname    string
		previousRun string
	}{
		{
			testname:    "withCaret",
			previousRun: fmt.Sprintf("%s^", appOneWithBuildTask),
		},
		{
			testname:    "withRunID",
			previousRun: "1",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.testname, func(t *testing.T) {
			redirectOutputToLogger(t)
			r := repotest.CreateBaurRepository(t, repotest.WithNewDB())
			r.CreateAppWithNoOutputs(t, appOneName)

			doInitDb(t)

			runCmd := newRunCmd()
			runCmd.run(&runCmd.Command, []string{appOneWithBuildTask})
			runCmd.run(&runCmd.Command, []string{appOneWithTestTask})

			assertExitCode(t, 0)

			diffInputsCmd := newDiffInputsCmd()
			diffInputsCmd.SetArgs([]string{"2", tc.previousRun})
			executeWithoutError(t, diffInputsCmd)
		})
	}
}

func TestCurrentInputsAgainstPreviousRunThatHasDifferentInputsReturnsExitCode2(t *testing.T) {
	testcases := []struct {
		testname    string
		previousRun string
	}{
		{
			testname:    "withCaret",
			previousRun: fmt.Sprintf("%s^", appOneWithBuildTask),
		},
		{
			testname:    "withRunID",
			previousRun: "1",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.testname, func(t *testing.T) {
			redirectOutputToLogger(t)
			r := repotest.CreateBaurRepository(t, repotest.WithNewDB())
			r.CreateAppWithNoOutputs(t, appOneName)

			doInitDb(t)

			runCmd := newRunCmd()
			runCmd.inputStr = "an_input"
			runCmd.run(&runCmd.Command, []string{appOneWithBuildTask})

			assertExitCode(t, 2)

			diffInputsCmd := newDiffInputsCmd()
			diffInputsCmd.SetArgs([]string{appOneWithBuildTask, tc.previousRun})
			executeWithoutError(t, diffInputsCmd)
		})
	}
}

func TestPreviousRunAgainstAnotherPreviousRunThatHasDifferentInputsReturnsExitCode2(t *testing.T) {
	testcases := []struct {
		testname    string
		previousRun string
	}{
		{
			testname:    "withCaret",
			previousRun: fmt.Sprintf("%s^", appOneWithBuildTask),
		},
		{
			testname:    "withRunID",
			previousRun: "1",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.testname, func(t *testing.T) {
			redirectOutputToLogger(t)
			r := repotest.CreateBaurRepository(t, repotest.WithNewDB())
			r.CreateAppWithNoOutputs(t, appOneName)

			doInitDb(t)

			runCmd := newRunCmd()
			runCmd.run(&runCmd.Command, []string{appOneWithBuildTask})
			runCmd.inputStr = "an_input"
			runCmd.run(&runCmd.Command, []string{appOneWithTestTask})

			assertExitCode(t, 2)

			diffInputsCmd := newDiffInputsCmd()
			diffInputsCmd.SetArgs([]string{"2", tc.previousRun})
			executeWithoutError(t, diffInputsCmd)
		})
	}
}

// Different apps will always return exit code 2 because their .app.toml files differ
func TestDifferentAppsReturnExitCode2(t *testing.T) {
	redirectOutputToLogger(t)
	r := repotest.CreateBaurRepository(t, repotest.WithNewDB())
	r.CreateAppWithNoOutputs(t, appOneName)
	r.CreateAppWithNoOutputs(t, appTwoName)

	doInitDb(t)

	assertExitCode(t, 2)

	diffInputsCmd := newDiffInputsCmd()
	diffInputsCmd.SetArgs([]string{appOneWithBuildTask, appTwoWithBuildTask})
	executeWithoutError(t, diffInputsCmd)
}

func TestDifferencesOutputWithCorrectState(t *testing.T) {
	redirectOutputToLogger(t)
	r := repotest.CreateBaurRepository(t, repotest.WithNewDB())
	r.CreateAppWithNoOutputs(t, appOneName)
	fileName := "diff_test.txt"

	doInitDb(t)

	originalDigest := r.WriteAdditionalFileContents(t, appOneName, fileName, "original")
	runCmd := newRunCmd()
	runCmd.inputStr = "run_one"
	runCmd.run(&runCmd.Command, []string{appOneWithBuildTask})

	newDigest := r.WriteAdditionalFileContents(t, appOneName, fileName, "new")
	runCmd.inputStr = "run_two"
	runCmd.run(&runCmd.Command, []string{appOneWithBuildTask})

	exitFunc = func(code int) {}

	stdoutBuf, _ := interceptCmdOutput(t)

	diffInputsCmd := newDiffInputsCmd()
	diffInputsCmd.csv = true
	diffInputsCmd.SetArgs([]string{fmt.Sprintf("%s^^", appOneWithBuildTask), fmt.Sprintf("%s^", appOneWithBuildTask)})
	executeWithoutError(t, diffInputsCmd)

	expectedOutput := [][]string{
		{"D", filepath.FromSlash("app_one/diff_test.txt"), originalDigest.String(), newDigest.String()},
		{"-", "string:run_one", "sha384:95e52b4c9863a13d596d34df980988cb78bea9ec3381ba981e1656a84cc1c7456f6830bca0e8931be5f0f48593cb5d06", ""},
		{"+", "string:run_two", "", "sha384:f3d5e46502641c5591563a0d3157f19a9739616f07bdb4bbc0285cb0a12bd511c026db94f12c719378a20d0ffe85090e"},
	}

	actualOutput, err := csv.NewReader(stdoutBuf).ReadAll()
	require.NoError(t, err)

	assert.ElementsMatch(t, expectedOutput, actualOutput)
}

func executeWithoutError(t *testing.T, cmd *diffInputsCmd) {
	err := cmd.Execute()
	assert.Nil(t, err)
}
