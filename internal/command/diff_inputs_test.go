// +build dbtest

package command

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

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
			testname: "withAppWildCard",
			appTask:  "*.task",
		},
		{
			testname: "withTaskWildCard",
			appTask:  "app.*",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.testname, func(t *testing.T) {
			r := repotest.CreateBaurRepository(t)
			r.CreateAppWithNoOutputs(t, appOneName)

			diffInputsCmd := newDiffInputsCmd()
			diffInputsCmd.SetArgs([]string{tc.appTask, "app.task"})
			err := diffInputsCmd.Execute()

			assert.EqualError(t, err, fmt.Sprintf("%s contains a wild card character, wild card characters are not allowed", tc.appTask))
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
			r := repotest.CreateBaurRepository(t)
			r.CreateAppWithNoOutputs(t, appOneName)

			diffInputsCmd := newDiffInputsCmd()
			diffInputsCmd.SetArgs([]string{tc.appTask, "app.task"})
			err := diffInputsCmd.Execute()

			assert.EqualError(t, err, fmt.Sprintf("%s does not specify a task or task-run ID", tc.appTask))
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
	r := repotest.CreateBaurRepository(t)
	r.CreateAppWithNoOutputs(t, appOneName)

	assertExitCode(t, 1)

	diffInputsCmd := newDiffInputsCmd()
	diffInputsCmd.SetArgs([]string{appOneWithBuildTask, appOneWithBuildTask})
	executeWithoutError(t, diffInputsCmd)
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
	r := repotest.CreateBaurRepository(t, repotest.WithNewDB())
	r.CreateAppWithNoOutputs(t, appOneName)
	r.CreateAppWithNoOutputs(t, appTwoName)

	doInitDb(t)

	runCmd := newRunCmd()
	runCmd.run(&runCmd.Command, []string{appTwoName})

	assertExitCode(t, 2)

	diffInputsCmd := newDiffInputsCmd()
	diffInputsCmd.SetArgs([]string{appOneWithBuildTask, appTwoWithBuildTask})
	executeWithoutError(t, diffInputsCmd)
}

func executeWithoutError(t *testing.T, cmd *diffInputsCmd) {
	err := cmd.Execute()
	assert.Nil(t, err)
}
