//go:build dbtest
// +build dbtest

package command

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v3/internal/testutils/dbtest"
	"github.com/simplesurance/baur/v3/internal/testutils/repotest"
)

func TestStatusTaskSpecArgParsing(t *testing.T) {
	initTest(t)

	repoDir := filepath.Join(testdataDir, "multitasks")
	err := os.Chdir(repoDir)
	require.NoError(t, err)

	dbURL, err := dbtest.CreateDB(dbtest.UniqueDBName())
	require.NoError(t, err)

	t.Setenv(envVarPSQLURL, dbURL)

	runInitDb(t)

	type testcase struct {
		name            string
		taskRunArg      []string
		expectedTaskIDs []string
		preRun          func(t *testing.T)
	}

	testcases := []*testcase{
		{
			name:       "wildcard",
			taskRunArg: []string{"*"},
			expectedTaskIDs: []string{
				"app1.build",
				"app1.check",
				"app1.test",
				"app2.build",
				"app2.check",
				"app2.test",
				"app3.build",
				"app3.check",
				"app3.test",
				"app4.compile",
				"app4.lint",
			},
		},
		{
			name:       "appWildcard",
			taskRunArg: []string{"app2.*"},
			expectedTaskIDs: []string{
				"app2.build",
				"app2.check",
				"app2.test",
			},
		},
		{
			name:       "AppnameWildcardAndTaskName",
			taskRunArg: []string{"*.build"},
			expectedTaskIDs: []string{
				"app1.build",
				"app2.build",
				"app3.build",
			},
		},

		{
			name:       "appAndTaskWildcard",
			taskRunArg: []string{"*.*"},
			expectedTaskIDs: []string{
				"app1.build",
				"app1.check",
				"app1.test",
				"app2.build",
				"app2.check",
				"app2.test",
				"app3.build",
				"app3.check",
				"app3.test",
				"app4.compile",
				"app4.lint",
			},
		},

		{
			name:       "appName",
			taskRunArg: []string{"app1"},
			expectedTaskIDs: []string{
				"app1.build",
				"app1.check",
				"app1.test",
			},
		},
		{
			name:       "specificTaskSpec",
			taskRunArg: []string{"app4.lint"},
			expectedTaskIDs: []string{
				"app4.lint",
			},
		},

		{
			name:       "absPath",
			taskRunArg: []string{filepath.Join(repoDir, "dir2", "app4")},
			expectedTaskIDs: []string{
				"app4.compile",
				"app4.lint",
			},
		},
		{
			name:       "relPath",
			taskRunArg: []string{filepath.Join("dir2", "app4")},
			expectedTaskIDs: []string{
				"app4.compile",
				"app4.lint",
			},
		},
		{
			name:       "currentDirPath",
			taskRunArg: []string{"."},
			expectedTaskIDs: []string{
				"app3.build",
				"app3.check",
				"app3.test",
			},
			preRun: func(t *testing.T) {
				cwd, err := os.Getwd()
				require.NoError(t, err)

				require.NoError(t, os.Chdir("app3"))
				t.Cleanup(func() {
					require.NoError(t, os.Chdir(cwd))
				})
			},
		},
		{
			name:       "parentDirPath",
			taskRunArg: []string{".."},
			expectedTaskIDs: []string{
				"app3.build",
				"app3.check",
				"app3.test",
			},
			preRun: func(t *testing.T) {
				childDir := filepath.Join(repoDir, "app3", uuid.New().String())
				require.NoError(t, os.Mkdir(childDir, 0700))

				t.Cleanup(func() {
					require.NoError(t, os.RemoveAll(childDir))
				})

				cwd, err := os.Getwd()
				require.NoError(t, err)

				require.NoError(t, os.Chdir(childDir))
				t.Cleanup(func() {
					require.NoError(t, os.Chdir(cwd))
				})
			},
		},

		{
			name: "multipleSpecs",
			taskRunArg: []string{
				"app1",
				"app2.check",
				"app3.*",
				filepath.Join("dir2", "app4"),
			},
			expectedTaskIDs: []string{
				"app1.build",
				"app1.check",
				"app1.test",
				"app2.check",
				"app3.build",
				"app3.check",
				"app3.test",
				"app4.compile",
				"app4.lint",
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			initTest(t)

			if tc.preRun != nil {
				tc.preRun(t)
			}

			statusCmd := newStatusCmd()
			statusCmd.SetArgs(tc.taskRunArg)
			statusOut := baurCSVStatusCmd(t, statusCmd)
			assert.Len(t, statusOut, len(tc.expectedTaskIDs))
			for _, line := range statusOut {
				assert.Contains(t, tc.expectedTaskIDs, line.taskID)
			}
		})
	}
}

func TestStatusCombininingFieldAndStatusParameters(t *testing.T) {
	initTest(t)

	r := repotest.CreateBaurRepository(t, repotest.WithNewDB())
	app := r.CreateSimpleApp(t)

	runInitDb(t)
	stdoutBuf, _ := interceptCmdOutput(t)
	statusCmd := newStatusCmd()
	statusCmd.SetArgs([]string{"-f", "task-id", "-s", "pending"})
	err := statusCmd.Execute()
	require.NoError(t, err)

	require.Contains(t, stdoutBuf.String(), app.Name)
}
