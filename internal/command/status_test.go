// +build dbtest

package command

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v1/internal/testutils/repotest"
)

func TestStatusArgs(t *testing.T) {
	r := repotest.CreateBaurRepository(t, repotest.WithNewDB())
	app := r.CreateSimpleApp(t)
	taskSpec := fmt.Sprintf("%s.%s", app.Name, app.Tasks[0].Name)
	runInitDb(t)
	statusCmd := newStatusCmd()

	type testcase struct {
		name       string
		taskRunArg string
	}

	testcases := []*testcase{
		{
			name:       "appName",
			taskRunArg: app.Name,
		},
		{
			name:       "wildcard",
			taskRunArg: "*",
		},
		{
			name:       "taskSpec",
			taskRunArg: fmt.Sprintf("%s.%s", app.Name, app.Tasks[0].Name),
		},
		{
			name:       "taskSpecTaskWildcard",
			taskRunArg: fmt.Sprintf("%s.%s", app.Name, "*"),
		},
		{
			name:       "taskSpecAppWildcard",
			taskRunArg: fmt.Sprintf("%s.%s", "*", app.Tasks[0].Name),
		},
		{
			name:       "absPath",
			taskRunArg: filepath.Dir(app.FilePath()),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			initTest(t)
			stdoutBuf, _ := interceptCmdOutput(t)

			statusCmd.Command.Run(&statusCmd.Command, []string{tc.taskRunArg})
			assert.Contains(t, stdoutBuf.String(), taskSpec)
		})
	}

	t.Run("relPath", func(t *testing.T) {
		initTest(t)
		stdoutBuf, _ := interceptCmdOutput(t)

		appDir := filepath.Dir(app.FilePath())
		appRelPath, err := filepath.Rel(r.Dir, appDir)
		require.NoError(t, err)

		statusCmd.Command.Run(&statusCmd.Command, []string{appRelPath})
		assert.Contains(t, stdoutBuf.String(), taskSpec)
	})

	t.Run("currentDirPath", func(t *testing.T) {
		initTest(t)
		stdoutBuf, _ := interceptCmdOutput(t)

		appDir := filepath.Dir(app.FilePath())
		require.NoError(t, os.Chdir(appDir))
		statusCmd.Command.Run(&statusCmd.Command, []string{"."})
		assert.Contains(t, stdoutBuf.String(), taskSpec)
	})

	t.Run("parentDirPath", func(t *testing.T) {
		initTest(t)
		stdoutBuf, _ := interceptCmdOutput(t)

		appDir := filepath.Dir(app.FilePath())
		childDir := filepath.Join(appDir, uuid.New().String())

		require.NoError(t, os.Mkdir(childDir, 0700))
		require.NoError(t, os.Chdir(childDir))

		statusCmd.Command.Run(&statusCmd.Command, []string{".."})
		assert.Contains(t, stdoutBuf.String(), taskSpec)
	})
}

func TestStatusCombininingFieldAndStatusParameters(t *testing.T) {
	initTest(t)

	r := repotest.CreateBaurRepository(t, repotest.WithNewDB())
	app := r.CreateSimpleApp(t)

	runInitDb(t)
	stdoutBuf, _ := interceptCmdOutput(t)
	statusCmd := newStatusCmd()
	statusCmd.SetArgs([]string{"-f", "task-id", "-s", "pending"})
	statusCmd.Execute()

	require.Contains(t, stdoutBuf.String(), app.Name)
}
