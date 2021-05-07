// +build dbtest

package command

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v2/internal/testutils/repotest"
	"github.com/simplesurance/baur/v2/pkg/cfg"
)

func TestRunSimultaneously(t *testing.T) {
	initTest(t)

	r := repotest.CreateBaurRepository(t, repotest.WithNewDB(), repotest.WithKeepTmpDir())

	parallelTaskCnt := 6
	apps := make([]*cfg.App, parallelTaskCnt)

	var checkScript = []byte(`#!/usr/bin/env bash
set -eu -o pipefail

parallel_tasks="$1"
max_iter=20
process_found=0

for (( i=0; i< $parallel_tasks; i++ )); do
	for (( j=0; ; j++ )); do
		# [c] is needed to exclude the grep process itself from the result
		ps -s | grep -q "[c]heckscript${i}.sh" && {
			echo "task $i is running"
			break
		}

		if [ $j -eq $max_iter ]; then
			echo "task $i not running"
			exit 1
		fi

		sleep 0.5
	done
done

# sleep a bit to give all parallel running processes a chance to find each other
sleep 3
`)

	var tasks cfg.Tasks

	for i := 0; i < parallelTaskCnt; i++ {
		err := ioutil.WriteFile(
			filepath.Join(r.Dir, fmt.Sprintf("checkscript%d.sh", i)),
			checkScript,
			0755,
		)
		require.NoError(t, err)

		apps[i] = r.CreateAppWithNoOutputs(t, fmt.Sprintf("myapp-%d", i))

		tasks = append(tasks, &cfg.Task{
			Name: fmt.Sprintf("check%d", i),
			Command: []string{
				"bash",
				filepath.Join(r.Dir, fmt.Sprintf("checkscript%d.sh", i)),
				fmt.Sprint(parallelTaskCnt),
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

	_, _ = interceptCmdOutput(t)

	runCmdTest := newRunCmd()
	runCmdTest.SetArgs([]string{"-p", fmt.Sprint(parallelTaskCnt)})
	err = runCmdTest.Execute()
	require.NoError(t, err)
}
