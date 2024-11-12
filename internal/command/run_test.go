//go:build dbtest
// +build dbtest

package command

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v5/internal/testutils/fstest"
	"github.com/simplesurance/baur/v5/internal/testutils/gittest"
	"github.com/simplesurance/baur/v5/internal/testutils/repotest"
	"github.com/simplesurance/baur/v5/pkg/cfg"
)

func TestRunSimultaneously(t *testing.T) {
	initTest(t)

	r := repotest.CreateBaurRepository(t, repotest.WithNewDB())

	parallelTaskCnt := 6

	// checkScript is exected by the tasks, it logs the start and end
	// timestamp of the script to file.
	// The test checks if all task start/endtimestamps are in the same period.
	checkScript := []byte(`#!/usr/bin/env bash
set -veu -o pipefail

runtime_logfile="$1"

date +%s > "$runtime_logfile"
sleep 3
date +%s >> "$runtime_logfile"
`)

	var tasks cfg.Tasks

	var runtimeLogfiles []string

	for i := 0; i < parallelTaskCnt; i++ {
		err := os.WriteFile(
			filepath.Join(r.Dir, fmt.Sprintf("checkscript%d.sh", i)),
			checkScript,
			0o755,
		)
		require.NoError(t, err)

		logfile := filepath.Join(r.Dir, fmt.Sprintf("runtimelog-task-%d", i))
		runtimeLogfiles = append(runtimeLogfiles, logfile)

		tasks = append(tasks, &cfg.Task{
			Name: fmt.Sprintf("check%d", i),
			Command: []string{
				"bash",
				filepath.Join(r.Dir, fmt.Sprintf("checkscript%d.sh", i)),
				logfile,
			},
			Input: cfg.Input{
				Files: []cfg.FileInputs{
					{Paths: []string{".app.toml"}},
				},
			},
		})
	}

	appCfg := cfg.App{
		Name:  "testapp",
		Tasks: tasks,
	}

	err := appCfg.ToFile(filepath.Join(r.Dir, ".app.toml"))
	require.NoError(t, err)

	doInitDb(t)

	runCmd := newRunCmd()
	runCmd.SetArgs([]string{"-p", fmt.Sprint(parallelTaskCnt)})
	err = runCmd.Execute()
	require.NoError(t, err)

	// check if all tasks were running during same timeperiod,
	// we can not use ps because it's not possible on Windows to get the
	// cmdline of a running process via ps/tasklist, all parallel running
	// tasks only show up as "bash".

	type runtime struct {
		startTime int64
		endTime   int64
	}

	taskruntimes := make([]runtime, 0, len(runtimeLogfiles))
	for _, logfile := range runtimeLogfiles {
		content, err := os.ReadFile(logfile)
		require.NoError(t, err)
		lines := strings.Fields(string(content))
		require.Len(t, lines, 2, "%s file content: %q, expected to find 2 lines", logfile, string(content))
		startTime, err := strconv.ParseInt(lines[0], 10, 64)
		require.NoError(t, err)

		endTime, err := strconv.ParseInt(lines[1], 10, 64)
		require.NoError(t, err)
		taskruntimes = append(taskruntimes, runtime{startTime: startTime, endTime: endTime})
	}

	for i := 1; i < len(taskruntimes); i++ {
		require.GreaterOrEqual(t, taskruntimes[0].startTime+1, taskruntimes[i].startTime)
		require.LessOrEqual(t, taskruntimes[0].endTime-1, taskruntimes[i].endTime)
		t.Logf("task %d run in parallel with task 0: starttime %d >= %d, endtime %d <= %d",
			i,
			taskruntimes[i].startTime, taskruntimes[0].startTime, taskruntimes[i].endTime, taskruntimes[0].endTime,
		)
	}
}

func TestRunShowOutput(t *testing.T) {
	initTest(t)
	r := repotest.CreateBaurRepository(t, repotest.WithNewDB())

	scriptPath := filepath.Join(r.Dir, "script.sh")

	err := os.WriteFile(
		scriptPath, []byte(`#/usr/bin/env bash
echo "greetings from script.sh"
	`),
		0o755)
	require.NoError(t, err)

	appCfg := cfg.App{
		Name: "testapp",
		Tasks: cfg.Tasks{{
			Name:    "build",
			Command: []string{"bash", scriptPath},
			Input: cfg.Input{
				Files: []cfg.FileInputs{
					{Paths: []string{".app.toml"}},
				},
			},
		}},
	}

	err = appCfg.ToFile(filepath.Join(r.Dir, ".app.toml"))
	require.NoError(t, err)

	doInitDb(t)

	runCmdTest := newRunCmd()
	runCmdTest.SetArgs([]string{"-o"})
	_, stderr := interceptCmdOutput(t)

	err = runCmdTest.Execute()
	require.NoError(t, err)

	require.Equal(t, 1, strings.Count(stderr.String(), "greetings from script.sh"))
}

func TestRunShowOutputOnErrorOutputIsPrintedOnce(t *testing.T) {
	initTest(t)
	r := repotest.CreateBaurRepository(t, repotest.WithNewDB())

	scriptPath := filepath.Join(r.Dir, "script.sh")

	err := os.WriteFile(
		scriptPath, []byte(`#/usr/bin/env bash
echo "I will fail!"
exit 1
	`),
		0o755)
	require.NoError(t, err)

	appCfg := cfg.App{
		Name: "testapp",
		Tasks: cfg.Tasks{{
			Name:    "build",
			Command: []string{"bash", scriptPath},
			Input: cfg.Input{
				Files: []cfg.FileInputs{
					{Paths: []string{".app.toml"}},
				},
			},
		}},
	}

	err = appCfg.ToFile(filepath.Join(r.Dir, ".app.toml"))
	require.NoError(t, err)

	doInitDb(t)

	runCmdTest := newRunCmd()
	runCmdTest.SetArgs([]string{"-o"})
	_, stderr := interceptCmdOutput(t)

	oldExitFunc := exitFunc
	var exitCode int
	exitFunc = func(code int) {
		exitCode = code
	}
	t.Cleanup(func() {
		exitFunc = oldExitFunc
	})

	err = runCmdTest.Execute()
	require.NoError(t, err)
	require.Equal(t, 1, exitCode)

	require.Equal(t, 1, strings.Count(stderr.String(), "I will fail!"))
}

func createEnvVarTestApp(t *testing.T, appName, taskName string, envVarsCfg []cfg.EnvVarsInputs) {
	r := repotest.CreateBaurRepository(t, repotest.WithNewDB())
	scriptPath := filepath.Join(r.Dir, "script.sh")
	err := os.WriteFile(
		scriptPath, []byte(`#/usr/bin/env bash
exit 0
	`),
		0o755)
	require.NoError(t, err)
	doInitDb(t)

	appCfg := cfg.App{
		Name: appName,
		Tasks: cfg.Tasks{{
			Name:    taskName,
			Command: []string{"bash", scriptPath},
			Input: cfg.Input{
				Files:                []cfg.FileInputs{{Paths: []string{scriptPath}}},
				EnvironmentVariables: envVarsCfg,
			},
		}},
	}
	err = appCfg.ToFile(filepath.Join(r.Dir, ".app.toml"))
	require.NoError(t, err)
}

func TestEnvVarInput_Required(t *testing.T) {
	const envVarName = "BAUR_TEST_ENV_VAR"
	const appName = "myapp"
	const taskName = "build"

	initTest(t)

	envVarInputs := []cfg.EnvVarsInputs{
		{Names: []string{envVarName}},
	}
	createEnvVarTestApp(t, appName, taskName, envVarInputs)

	t.Run("run_fails_when_required_env_var_is_undefined", func(t *testing.T) {
		var exitCode int

		initTest(t)

		interceptExitCode(t, &exitCode)
		_, stderr := interceptCmdOutput(t)

		require.NoError(t, newRunCmd().Execute())
		require.Equal(t, 1, exitCode, "command did not exit with code 1")
		assert.Contains(t,
			stderr.String(),
			fmt.Sprintf("environment variable %q is undefined", envVarName),
		)
	})

	for _, envVarval := range []string{"", "hello"} {
		t.Run(fmt.Sprintf("env_var_val_%q", envVarval), func(t *testing.T) {
			initTest(t)

			stdout, _ := interceptCmdOutput(t)

			t.Setenv(envVarName, envVarval)
			require.NoError(t, newRunCmd().Execute())
			outStr := stdout.String()
			assert.Contains(
				t,
				outStr,
				fmt.Sprintf("%s.%s: run stored in database with ID", appName, taskName),
			)

			stdout, _ = interceptCmdOutput(t)
			statusCmd := newStatusCmd()
			statusCmd.SetArgs([]string{
				"--format=csv", "-q", "-f", "run-id", fmt.Sprintf("%s.%s", appName, taskName),
			},
			)
			require.NoError(t, statusCmd.Execute())
			runID := strings.TrimSpace(stdout.String())

			stdout, stderr := interceptCmdOutput(t)
			lsInputsCmd := newLsInputsCmd()
			lsInputsCmd.SetArgs([]string{"--format=csv", runID})
			require.NoError(t, lsInputsCmd.Execute())
			assert.Contains(t, stdout.String(), "$"+envVarName, "env var is missing in 'ls inputs' output")

			t.Setenv(envVarName, "rerunplz"+t.Name())
			require.NoError(t, newRunCmd().Execute())
			t.Log(stdout)
			t.Log(stderr)
			assert.Contains(
				t,
				stdout.String(),
				fmt.Sprintf("%s.%s: run stored in database with ID", appName, taskName),
			)
		})
	}
}

func TestEnvVarInput_Optional(t *testing.T) {
	const envVarName = "BAUR_b_TEST_ENV_VAR"
	const appName = "myapp"
	const taskName = "build"

	envVarInputs := []cfg.EnvVarsInputs{
		{
			Names:    []string{envVarName},
			Optional: true,
		},
	}
	createEnvVarTestApp(t, appName, taskName, envVarInputs)
	t.Run("run_succeeds_with_optional_undefined_env_var", func(t *testing.T) {
		initTest(t)

		require.NotPanics(t, func() {
			require.NoError(t, newRunCmd().Execute())
		})
	})

	t.Run("ls_inputs_succeeds_with_optional_undefined_env_var", func(t *testing.T) {
		initTest(t)

		lsInputsCmd := newLsInputsCmd()
		lsInputsCmd.SetArgs([]string{fmt.Sprintf("%s.%s", appName, taskName)})

		require.NotPanics(t, func() {
			require.NoError(t, lsInputsCmd.Execute())
		})
	})

	t.Run("status_succeeds_with_optional_undefined_env_var", func(t *testing.T) {
		initTest(t)

		lsInputsCmd := newLsInputsCmd()
		lsInputsCmd.SetArgs([]string{fmt.Sprintf("%s.%s", appName, taskName)})

		require.NotPanics(t, func() {
			require.NoError(t, newStatusCmd().Execute())
		})
	})
}

func TestRunFailsWhenGitWorktreeIsDirty(t *testing.T) {
	initTest(t)

	r := repotest.CreateBaurRepository(t, repotest.WithNewDB())
	gittest.CreateRepository(t, r.Dir)
	r.CreateSimpleApp(t)
	fname := "untrackedFile"
	fstest.WriteToFile(t, []byte("hello"), filepath.Join(r.Dir, fname))

	_, stderrBuf := interceptCmdOutput(t)
	runCmd := newRunCmd()
	runCmd.SetArgs([]string{"--" + flagNameRequireCleanGitWorktree})
	require.Panics(t, func() { require.NoError(t, runCmd.Execute()) })

	require.Contains(t, stderrBuf.String(), fname)
	require.Contains(t, stderrBuf.String(), "expecting only tracked unmodified files")
}
