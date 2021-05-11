// +build dbtest

package command

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v2/internal/testutils/repotest"
	"github.com/simplesurance/baur/v2/pkg/cfg"
)

func TestRunSimultaneously(t *testing.T) {
	initTest(t)

	r := repotest.CreateBaurRepository(t, repotest.WithNewDB())

	parallelTaskCnt := 6

	// checkScript is exected by the tasks, it logs the start and end
	// timestamp of the script to file.
	// The test checks if all task start/endtimestamps are in the same period.
	var checkScript = []byte(`#!/usr/bin/env bash
set -veu -o pipefail

runtime_logfile="$1"

date +%s > "$runtime_logfile"
sleep 3
date +%s >> "$runtime_logfile"
`)

	var tasks cfg.Tasks

	var runtimeLogfiles []string

	for i := 0; i < parallelTaskCnt; i++ {
		err := ioutil.WriteFile(
			filepath.Join(r.Dir, fmt.Sprintf("checkscript%d.sh", i)),
			checkScript,
			0755,
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

	runCmdTest := newRunCmd()
	runCmdTest.SetArgs([]string{"-p", fmt.Sprint(parallelTaskCnt)})
	err = runCmdTest.Execute()
	require.NoError(t, err)

	// check if all tasks were running during same timeperiod,
	// we can not use ps because it's not possible on Windows to get the
	// cmdline of a running process via ps/tasklist, all parallel running
	// tasks only show up as "bash".

	type runtime struct {
		startTime int64
		endTime   int64
	}
	var taskruntimes []runtime
	for _, logfile := range runtimeLogfiles {
		content, err := ioutil.ReadFile(logfile)
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
		require.GreaterOrEqual(t, taskruntimes[0].startTime, taskruntimes[i].startTime)
		require.LessOrEqual(t, taskruntimes[0].endTime, taskruntimes[i].endTime)
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

	err := ioutil.WriteFile(
		scriptPath, []byte(`#/usr/bin/env bash
echo "greetings from script.sh"
	`),
		0755)
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

	doInitDb(t)

	runCmdTest := newRunCmd()
	runCmdTest.SetArgs([]string{"-o"})
	stdout, _ := interceptCmdOutput(t)

	err = runCmdTest.Execute()
	require.NoError(t, err)

	require.Equal(t, 1, strings.Count(stdout.String(), "greetings from script.sh"))
}

func TestRunShowOutputOnErrorOutputIsPrintedOnce(t *testing.T) {
	initTest(t)
	r := repotest.CreateBaurRepository(t, repotest.WithNewDB())

	scriptPath := filepath.Join(r.Dir, "script.sh")

	err := ioutil.WriteFile(
		scriptPath, []byte(`#/usr/bin/env bash
echo "I will fail!"
exit 1
	`),
		0755)
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

	doInitDb(t)

	runCmdTest := newRunCmd()
	runCmdTest.SetArgs([]string{"-o"})
	stdout, _ := interceptCmdOutput(t)

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

	require.Equal(t, 1, strings.Count(stdout.String(), "I will fail!"))
}
