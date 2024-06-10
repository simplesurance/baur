//go:build dbtest
// +build dbtest

package command

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v4/internal/fs"
	"github.com/simplesurance/baur/v4/internal/testutils/fstest"
	"github.com/simplesurance/baur/v4/internal/testutils/gittest"
	"github.com/simplesurance/baur/v4/internal/testutils/repotest"
	"github.com/simplesurance/baur/v4/pkg/baur"
	"github.com/simplesurance/baur/v4/pkg/cfg"
)

func writeTaskInfoCheckScript(t *testing.T, uuid, scriptPath, taskInfoCpDest string) {
	checkScript := fmt.Sprintf(`
	set -x -eu -o pipefail
	# uuid: %s

	fatal() {
		echo "$@" > 1
		exit 1
	}

	[[ ! -v BUILD_TASK_INFO ]] && {
		fatal "BUILD_TASK_INFO environment variable is not set."
	}
	[ -z "$BUILD_TASK_INFO" ] && {
		fatal "BUILD_TASK_INFO environment varible is set but empty"
	}
	[ ! -e "$BUILD_TASK_INFO" ] && {
		fatal "\$BUILD_TASK_INFO ($BUILD_TASK_INFO) file does not exist"
	}
	cp "$BUILD_TASK_INFO" "%s"
	`, uuid, taskInfoCpDest)
	fstest.WriteToFile(t, []byte(checkScript), scriptPath)
}

func unmarshalTaskInfoFile(t *testing.T, path string) *baur.TaskInfoFile {
	var taskInfo baur.TaskInfoFile
	taskInfoRaw := fstest.ReadFile(t, path)
	t.Log(string(taskInfoRaw))
	require.NoError(t, json.Unmarshal(taskInfoRaw, &taskInfo))
	return &taskInfo
}

func TestRun_TaskInfoFileContent(t *testing.T) {
	initTest(t)
	outdestdir := t.TempDir()

	r := repotest.CreateBaurRepository(t, repotest.WithNewDB())
	gittest.CreateRepository(t, r.Dir)
	app := cfg.App{
		Name: "app",
		Tasks: []*cfg.Task{
			{
				Name:    "build",
				Command: []string{"bash", "./build.sh"},
				Output: cfg.Output{
					File: []cfg.FileOutput{{
						Path: "output",
						FileCopy: []cfg.FileCopy{{
							Path: filepath.Join(outdestdir, "out"),
						}},
					}},
				},
				Input: cfg.Input{
					Files: []cfg.FileInputs{{
						Paths: []string{"build.sh"},
					}},
				},
			},
			{
				Name:    "check",
				Command: []string{"bash", "check.sh"},
				Input: cfg.Input{
					TaskInfos: []cfg.TaskInfo{{
						TaskName:   "build",
						EnvVarName: "BUILD_TASK_INFO",
					}},
					Files: []cfg.FileInputs{{
						Paths: []string{"check.sh"},
					}},
				},
			},
		},
	}

	appdir := filepath.Join(r.Dir, "app")
	fstest.MkdirAll(t, appdir)
	require.NoError(t, app.ToFile(filepath.Join(appdir, ".app.toml")))

	const buildScript = `echo hello > output`
	fstest.WriteToFile(t, []byte(buildScript), filepath.Join(appdir, "build.sh"))

	taskInfoResultPath := filepath.Join(t.TempDir(), "taskinfo.json")

	checkScriptPath := filepath.Join(appdir, "check.sh")
	writeTaskInfoCheckScript(t, uuid.NewString(), checkScriptPath, taskInfoResultPath)
	doInitDb(t)

	t.Run("dependent-build-run-does-not-exist", func(t *testing.T) {
		t.Cleanup(func() { require.NoError(t, os.Remove(taskInfoResultPath)) })

		runCmd := newRunCmd()
		runCmd.SetArgs([]string{"app.check"})
		require.NotPanics(t, func() { require.NoError(t, runCmd.Execute()) })

		taskInfo := unmarshalTaskInfoFile(t, taskInfoResultPath)

		require.Equal(t, appdir, taskInfo.AppDir)
		require.NotEmpty(t, taskInfo.TotalInputDigest)
		require.Len(t, taskInfo.Outputs, 1)
		require.Equal(t, filepath.Join(outdestdir, "out"), taskInfo.Outputs[0].URI)
		require.False(t, fs.FileExists(taskInfo.Outputs[0].URI))
	})

	t.Run("dependent-build-run-exists", func(t *testing.T) {
		t.Cleanup(func() { require.NoError(t, os.Remove(taskInfoResultPath)) })
		runCmd := newRunCmd()
		runCmd.SetArgs([]string{"app.build"})
		require.NotPanics(t, func() { require.NoError(t, runCmd.Execute()) })

		// change uuid in script to enforce change + task run
		writeTaskInfoCheckScript(t, uuid.NewString(), checkScriptPath, taskInfoResultPath)

		runCmd = newRunCmd()
		runCmd.SetArgs([]string{"app.check"})
		require.NotPanics(t, func() { require.NoError(t, runCmd.Execute()) })

		taskInfo := unmarshalTaskInfoFile(t, taskInfoResultPath)

		require.Equal(t, appdir, taskInfo.AppDir)
		require.NotEmpty(t, taskInfo.TotalInputDigest)
		require.Len(t, taskInfo.Outputs, 1)
		require.Equal(t, filepath.Join(outdestdir, "out", "output"), taskInfo.Outputs[0].URI)
		require.True(t, fs.FileExists(taskInfo.Outputs[0].URI))

		require.NoError(t, os.Remove(taskInfo.Outputs[0].URI))
	})

	t.Run("with-input-str", func(t *testing.T) {
		t.Cleanup(func() { require.NoError(t, os.Remove(taskInfoResultPath)) })
		// change uuid in script to enforce change + task run
		writeTaskInfoCheckScript(t, uuid.NewString(), checkScriptPath, taskInfoResultPath)

		runCmd := newRunCmd()
		runCmd.SetArgs([]string{"app.check"})
		require.NotPanics(t, func() { require.NoError(t, runCmd.Execute()) })

		taskInfo := unmarshalTaskInfoFile(t, taskInfoResultPath)
		buildDigestNoInputStr := taskInfo.TotalInputDigest
		require.NoError(t, os.Remove(taskInfoResultPath))

		runCmd.inputStr = []string{"abc"}
		require.NotPanics(t, func() { require.NoError(t, runCmd.Execute()) })
		taskInfo = unmarshalTaskInfoFile(t, taskInfoResultPath)
		digestWithInputStr := taskInfo.TotalInputDigest

		require.NotEqual(t, buildDigestNoInputStr, digestWithInputStr)
	})
}

func TestLsInputsOutputs_TaskInfo(t *testing.T) {
	initTest(t)

	r := repotest.CreateBaurRepository(t, repotest.WithNewDB())
	gittest.CreateRepository(t, r.Dir)
	app := cfg.App{
		Name: "app",
		Tasks: []*cfg.Task{
			{
				Name:    "build",
				Command: []string{"bash", "-c", "true"},
			},
			{
				Name:    "check",
				Command: []string{"bash", "-c", "true"},
				Input: cfg.Input{
					TaskInfos: []cfg.TaskInfo{{
						TaskName:   "build",
						EnvVarName: "BUILD_TASK_INFO",
					}},
				},
			},
		},
	}

	appdir := filepath.Join(r.Dir, "app")
	fstest.MkdirAll(t, appdir)
	require.NoError(t, app.ToFile(filepath.Join(appdir, ".app.toml")))
	doInitDb(t)
	stdout, _ := interceptCmdOutput(t)

	lsInputsCmd := newLsInputsCmd()
	lsInputsCmd.SetArgs([]string{"app.check"})
	require.NotPanics(t, func() { require.NoError(t, lsInputsCmd.Execute()) })
	require.Contains(t, stdout.String(), "task: app.build")

	runCmd := newRunCmd()
	stdout, _ = interceptCmdOutput(t)
	runCmd.SetArgs([]string{"app.check"})
	require.NotPanics(t, func() { require.NoError(t, runCmd.Execute()) })
	t.Log(stdout.String())
	require.Contains(t, stdout.String(), "app.check: run stored in database with ID 1")

	stdout, _ = interceptCmdOutput(t)
	lsInputsCmd.SetArgs([]string{"1"})
	require.NotPanics(t, func() { require.NoError(t, lsInputsCmd.Execute()) })
	require.Contains(t, stdout.String(), "task: app.build")
}
