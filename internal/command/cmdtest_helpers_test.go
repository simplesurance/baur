//go:build dbtest

package command

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v5/internal/testutils/repotest"
	"github.com/simplesurance/baur/v5/pkg/baur"
	"github.com/simplesurance/baur/v5/pkg/cfg"
	"github.com/simplesurance/baur/v5/pkg/storage"
)

const testAppName = "svc"

// shTask returns a task that runs script via "sh -c".
// The only input of the task is the app config file and the files in extraInputs
// (paths relative to the app directory).
func shTask(name, script string, outputs []cfg.FileOutput, extraInputs ...string) *cfg.Task {
	return &cfg.Task{
		Name:    name,
		Command: []string{"sh", "-c", script},
		Input: cfg.Input{
			Files: []cfg.FileInputs{
				{Paths: append([]string{".app.toml"}, extraInputs...)},
			},
		},
		Output: cfg.Output{File: outputs},
	}
}

// newRepoWithTasks creates a baur repository with a new database and the
// schema, containing one app named [testAppName] with the passed tasks.
// The app directory is <repo>/svc.
func newRepoWithTasks(t *testing.T, tasks ...*cfg.Task) *repotest.Repo {
	t.Helper()

	initTest(t)
	// the environment variable has priority over the repository config,
	// unset it to prevent that commands use a database outside of the test
	t.Setenv(envVarPSQLURL, "")

	r := repotest.CreateBaurRepository(t, repotest.WithNewDB())

	appDir := filepath.Join(r.Dir, testAppName)
	require.NoError(t, os.Mkdir(appDir, 0o755))

	app := &cfg.App{Name: testAppName, Tasks: tasks}
	require.NoError(t, app.ToFile(filepath.Join(appDir, baur.AppCfgFile)))
	r.AppCfgs = append(r.AppCfgs, app)

	runInitDb(t)

	return r
}

// storedTaskRunIDs returns the IDs of all task runs stored in the database of r.
func storedTaskRunIDs(t *testing.T, r *repotest.Repo) []int {
	t.Helper()

	clt, err := newStorageClient(r.Cfg.Database.PGSQLURL)
	require.NoError(t, err)

	defer clt.Close()

	var ids []int

	err = clt.TaskRuns(t.Context(), nil, nil, storage.NoLimit, func(tr *storage.TaskRunWithID) error {
		ids = append(ids, tr.ID)
		return nil
	})
	if errors.Is(err, storage.ErrNotExist) {
		return nil
	}

	require.NoError(t, err)

	return ids
}

// releaseExists returns true if a release named name exists in the database of r.
func releaseExists(t *testing.T, r *repotest.Repo, name string) bool {
	t.Helper()

	clt, err := newStorageClient(r.Cfg.Database.PGSQLURL)
	require.NoError(t, err)

	defer clt.Close()

	exists, err := clt.ReleaseExists(t.Context(), name)
	require.NoError(t, err)

	return exists
}

// runTasks executes "baur run" with args and expects it to exit with
// expectedExitCode.
func runTasks(t *testing.T, expectedExitCode int, args ...string) {
	t.Helper()

	cmd := newRunCmd()
	cmd.SetArgs(args)
	execCheck(t, cmd, expectedExitCode)
}
