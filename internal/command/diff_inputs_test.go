//go:build dbtest
// +build dbtest

package command

import (
	"encoding/csv"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v5/internal/testutils/repotest"
)

var (
	appOneName          = "app-one"
	appTwoName          = "app-two"
	buildTaskName       = "build"
	testTaskName        = "test"
	appOneWithBuildTask = fmt.Sprintf("%s.%s", appOneName, buildTaskName)
	appOneWithTestTask  = fmt.Sprintf("%s.%s", appOneName, testTaskName)
	appTwoWithBuildTask = fmt.Sprintf("%s.%s", appTwoName, buildTaskName)
)

func doInitDb(t *testing.T) {
	t.Helper()

	t.Log("creating database schema")
	initDb(initDbCmd, nil)
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
			initTest(t)
			r := repotest.CreateBaurRepository(t)
			r.CreateAppWithNoOutputs(t, appOneName)

			diffInputsCmd := newDiffInputsCmd()
			diffInputsCmd.SetArgs(tc.args)
			err := diffInputsCmd.Execute()

			require.EqualError(t, err, fmt.Sprintf("accepts 2 args, received %d", len(tc.args)))
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
			initTest(t)
			r := repotest.CreateBaurRepository(t)
			r.CreateAppWithNoOutputs(t, appOneName)

			diffInputsCmd := newDiffInputsCmd()
			diffInputsCmd.SetArgs([]string{tc.appTask, "app.task"})
			err := diffInputsCmd.Execute()

			require.EqualError(t, err, fmt.Sprintf("invalid argument: \"%s\"", tc.appTask))
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
			initTest(t)
			r := repotest.CreateBaurRepository(t)
			r.CreateAppWithNoOutputs(t, appOneName)

			diffInputsCmd := newDiffInputsCmd()
			diffInputsCmd.SetArgs([]string{tc.appTask, "app.task"})
			err := diffInputsCmd.Execute()

			require.EqualError(t, err, fmt.Sprintf("invalid argument: \"%s\"", tc.appTask))
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
			initTest(t)
			r := repotest.CreateBaurRepository(t)
			r.CreateAppWithNoOutputs(t, appOneName)

			diffInputsCmd := newDiffInputsCmd()
			diffInputsCmd.SetArgs([]string{tc.appTask, appOneWithBuildTask})
			execCheck(t, diffInputsCmd, 1)
		})
	}
}

func TestCurrentInputsAgainstSameTaskCurrentInputsReturnsExitCode1(t *testing.T) {
	initTest(t)
	r := repotest.CreateBaurRepository(t)
	r.CreateAppWithNoOutputs(t, appOneName)

	diffInputsCmd := newDiffInputsCmd()
	diffInputsCmd.SetArgs([]string{appOneWithBuildTask, appOneWithBuildTask})
	_, stderrBuf := interceptCmdOutput(t)
	execCheck(t, diffInputsCmd, 1)
	assert.Contains(t, stderrBuf.String(), fmt.Sprintf("%s and %s refer to the same task-run", appOneWithBuildTask, appOneWithBuildTask))
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
			initTest(t)
			r := repotest.CreateBaurRepository(t, repotest.WithNewDB())
			r.CreateAppWithNoOutputs(t, appOneName)

			doInitDb(t)

			runCmd := newRunCmd()
			runCmd.run(&runCmd.Command, []string{appOneWithBuildTask})

			diffInputsCmd := newDiffInputsCmd()
			diffInputsCmd.SetArgs([]string{appOneWithBuildTask, tc.run})
			execCheck(t, diffInputsCmd, 1)
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
			initTest(t)
			r := repotest.CreateBaurRepository(t, repotest.WithNewDB())
			r.CreateAppWithNoOutputs(t, appOneName)

			doInitDb(t)

			runCmd := newRunCmd()
			runCmd.run(&runCmd.Command, []string{appOneWithBuildTask})

			diffInputsCmd := newDiffInputsCmd()
			diffInputsCmd.SetArgs([]string{appOneWithBuildTask, tc.previousRun})
			execCheck(t, diffInputsCmd, 0)
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
			initTest(t)
			r := repotest.CreateBaurRepository(t, repotest.WithNewDB())
			r.CreateAppWithNoOutputs(t, appOneName)

			doInitDb(t)

			runCmd := newRunCmd()
			runCmd.run(&runCmd.Command, []string{appOneWithBuildTask})
			runCmd.run(&runCmd.Command, []string{appOneWithTestTask})

			diffInputsCmd := newDiffInputsCmd()
			diffInputsCmd.SetArgs([]string{"2", tc.previousRun})
			execCheck(t, diffInputsCmd, 0)
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
			initTest(t)
			r := repotest.CreateBaurRepository(t, repotest.WithNewDB())
			r.CreateAppWithNoOutputs(t, appOneName)

			doInitDb(t)

			runCmd := newRunCmd()
			runCmd.inputStr = []string{"an_input"}
			runCmd.run(&runCmd.Command, []string{appOneWithBuildTask})

			diffInputsCmd := newDiffInputsCmd()
			diffInputsCmd.SetArgs([]string{appOneWithBuildTask, tc.previousRun})
			execCheck(t, diffInputsCmd, 2)
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
			initTest(t)
			r := repotest.CreateBaurRepository(t, repotest.WithNewDB())
			r.CreateAppWithNoOutputs(t, appOneName)

			doInitDb(t)

			runCmd := newRunCmd()
			runCmd.run(&runCmd.Command, []string{appOneWithBuildTask})
			runCmd.inputStr = []string{"an_input"}
			runCmd.run(&runCmd.Command, []string{appOneWithTestTask})

			diffInputsCmd := newDiffInputsCmd()
			diffInputsCmd.SetArgs([]string{"2", tc.previousRun})
			execCheck(t, diffInputsCmd, 2)
		})
	}
}

// Different apps will always return exit code 2 because their .app.toml files differ
func TestDifferentAppsReturnExitCode2(t *testing.T) {
	initTest(t)
	r := repotest.CreateBaurRepository(t, repotest.WithNewDB())
	r.CreateAppWithNoOutputs(t, appOneName)
	r.CreateAppWithNoOutputs(t, appTwoName)

	doInitDb(t)

	diffInputsCmd := newDiffInputsCmd()
	diffInputsCmd.SetArgs([]string{appOneWithBuildTask, appTwoWithBuildTask})
	execCheck(t, diffInputsCmd, 2)
}

func TestDifferencesOutputWithCorrectState(t *testing.T) {
	initTest(t)
	r := repotest.CreateBaurRepository(t, repotest.WithNewDB())
	r.CreateAppWithNoOutputs(t, appOneName)
	fileName := "diff_test.txt"

	doInitDb(t)

	originalDigest := r.WriteAdditionalFileContents(t, appOneName, fileName, "original")
	runCmd := newRunCmd()
	runCmd.inputStr = []string{"run_one"}
	runCmd.run(&runCmd.Command, []string{appOneWithBuildTask})

	newDigest := r.WriteAdditionalFileContents(t, appOneName, fileName, "new")
	runCmd.inputStr = []string{"run_two"}
	runCmd.run(&runCmd.Command, []string{appOneWithBuildTask})

	stdoutBuf, _ := interceptCmdOutput(t)
	diffInputsCmd := newDiffInputsCmd()
	diffInputsCmd.format.Val = "csv"
	diffInputsCmd.quiet = true
	diffInputsCmd.SetArgs([]string{fmt.Sprintf("%s^^", appOneWithBuildTask), fmt.Sprintf("%s^", appOneWithBuildTask)})
	execCheck(t, diffInputsCmd, -1)

	expectedOutput := [][]string{
		{"D", filepath.FromSlash("app-one/diff_test.txt"), originalDigest.String(), newDigest.String()},
		{"-", "string:run_one", "sha384:95e52b4c9863a13d596d34df980988cb78bea9ec3381ba981e1656a84cc1c7456f6830bca0e8931be5f0f48593cb5d06", ""},
		{"+", "string:run_two", "", "sha384:f3d5e46502641c5591563a0d3157f19a9739616f07bdb4bbc0285cb0a12bd511c026db94f12c719378a20d0ffe85090e"},
	}

	actualOutput, err := csv.NewReader(stdoutBuf).ReadAll()
	require.NoError(t, err)

	assert.ElementsMatch(t, expectedOutput, actualOutput)
}

func TestInputStringsAreAppliedToFirstArg(t *testing.T) {
	initTest(t)
	r := repotest.CreateBaurRepository(t, repotest.WithNewDB())
	r.CreateAppWithNoOutputs(t, appOneName)
	r.CreateAppWithNoOutputs(t, appTwoName)

	doInitDb(t)

	stdoutBuf, _ := interceptCmdOutput(t)
	diffInputsCmd := newDiffInputsCmd()
	diffInputsCmd.format.Val = "csv"
	diffInputsCmd.quiet = true
	diffInputsCmd.inputStr = []string{"hello"}
	diffInputsCmd.SetArgs([]string{appOneWithBuildTask, appTwoWithBuildTask})
	execCheck(t, diffInputsCmd, 2)
	assert.Contains(t, stdoutBuf.String(), "-,string:hello,")

	runCmd := newRunCmd()
	runCmd.inputStr = []string{"hello"}
	runCmd.run(&runCmd.Command, []string{appOneWithBuildTask})

	// inputs are the same, diffInputsCmd.inputStr is set to hello and added to first target
	diffInputsCmd.SetArgs([]string{appOneWithBuildTask, "1"})
	execCheck(t, diffInputsCmd, 0)

	// inputs differ, diffInputsCmd.inputStr is set to hello and added to
	// first target which is the stored run, app in repo is missing
	// input-string
	stdoutBuf, _ = interceptCmdOutput(t)
	diffInputsCmd.SetArgs([]string{"1", appOneWithBuildTask})
	execCheck(t, diffInputsCmd, 2)
	assert.Contains(t, stdoutBuf.String(), "-,string:hello,")
}
