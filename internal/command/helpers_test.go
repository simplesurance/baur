package command

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v5/internal/command/flag"
	"github.com/simplesurance/baur/v5/internal/testutils/fstest"
	"github.com/simplesurance/baur/v5/internal/testutils/gittest"
	"github.com/simplesurance/baur/v5/internal/testutils/repotest"
	"github.com/simplesurance/baur/v5/internal/vcs/git"
	"github.com/simplesurance/baur/v5/pkg/baur"
	"github.com/simplesurance/baur/v5/pkg/cfg"
)

// expectExit runs fn and fails if it does not terminate the program via
// exitFunc with expectedCode. exitFunc must have been replaced via initTest().
func expectExit(t *testing.T, expectedCode int, fn func()) {
	t.Helper()

	defer func() {
		t.Helper()

		r := recover()
		info, ok := r.(*exitInfo)
		if !ok {
			if r != nil {
				panic(r)
			}

			t.Fatalf("function did not terminate the program, expected exit code %d", expectedCode)
		}

		assert.Equal(t, expectedCode, info.Code)
	}()

	fn()
}

// setGlobal sets *p to val and restores the old value when the test finished.
func setGlobal[T any](t *testing.T, p *T, val T) {
	t.Helper()

	old := *p
	*p = val

	t.Cleanup(func() { *p = old })
}

// newGitRepoWithApps creates a baur repository that is part of a git repository.
// It contains the apps "simpleApp" (tasks: build, check) and "appWithoutIO" (no tasks).
func newGitRepoWithApps(t *testing.T) *repotest.Repo {
	t.Helper()

	initTest(t)

	r := repotest.CreateBaurRepository(t)
	r.CreateSimpleApp(t)
	r.CreateAppWithoutTasks(t)
	gittest.CommitFilesToGit(t, r.Dir)

	return r
}

func TestSubStr(t *testing.T) {
	tests := []struct {
		in           string
		start, count int
		want         string
	}{
		{"abcdef", 0, 3, "abc"},
		{"abcdef", 2, 2, "cd"},
		{"abcdef", 4, 10, "ef"},
		{"abcdef", 6, 1, ""},
		{"abcdef", 10, 1, ""},
		{"abcdef", 0, 0, ""},
		{"", 0, 3, ""},
		{"äöü€x", 1, 3, "öü€"},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("%q[%d:%d]", tc.in, tc.start, tc.count), func(t *testing.T) {
			assert.Equal(t, tc.want, subStr(tc.in, tc.start, tc.count))
		})
	}
}

func TestMaxTaskIDLen(t *testing.T) {
	assert.Equal(t, 0, maxTaskIDLen(nil))
	assert.Equal(t, 10, maxTaskIDLen([]*baur.Task{{ID: "a.b"}, {ID: "app.build1"}, {ID: "x.y"}}))
}

func TestExitHelpers(t *testing.T) {
	initTest(t)
	_, stderrBuf := interceptCmdOutput(t)

	t.Run("nil error does not exit", func(t *testing.T) {
		assert.NotPanics(t, func() {
			exitOnErr(nil)
			exitOnErr(nil, "msg")
			exitOnErrf(nil, "msg %s", "x")
		})
		assert.Empty(t, stderrBuf.String())
	})

	t.Run("exitOnErr", func(t *testing.T) {
		expectExit(t, exitCodeError, func() { exitOnErr(errors.New("boom-1"), "context-1") })
		assert.Contains(t, stderrBuf.String(), "boom-1")
		assert.Contains(t, stderrBuf.String(), "context-1")
	})

	t.Run("exitOnErrf", func(t *testing.T) {
		expectExit(t, exitCodeError, func() { exitOnErrf(errors.New("boom-2"), "context %d", 2) })
		assert.Contains(t, stderrBuf.String(), "boom-2")
		assert.Contains(t, stderrBuf.String(), "context 2")
	})

	t.Run("fatal", func(t *testing.T) {
		expectExit(t, exitCodeError, func() { fatal("fatal-3") })
		assert.Contains(t, stderrBuf.String(), "fatal-3")
	})

	t.Run("fatalf", func(t *testing.T) {
		expectExit(t, exitCodeError, func() { fatalf("fatal-%d", 4) })
		assert.Contains(t, stderrBuf.String(), "fatal-4")
	})
}

func TestMustNewFormatter(t *testing.T) {
	initTest(t)
	hdrs := []string{"Name", "Value"}

	t.Run("csv", func(t *testing.T) {
		stdoutBuf, _ := interceptCmdOutput(t)

		f := mustNewFormatter(flag.FormatCSV, hdrs)
		mustWriteRow(f, "a", 1)
		mustWriteRow(f, "b", 2)
		require.NoError(t, f.Flush())

		recs, err := csv.NewReader(stdoutBuf).ReadAll()
		require.NoError(t, err)
		assert.Equal(t, [][]string{{"Name", "Value"}, {"a", "1"}, {"b", "2"}}, recs)
	})

	t.Run("plain", func(t *testing.T) {
		stdoutBuf, _ := interceptCmdOutput(t)

		f := mustNewFormatter(flag.FormatPlain, hdrs)
		mustWriteRow(f, "a", 1)
		require.NoError(t, f.Flush())

		lines := strings.Split(strings.TrimSpace(stdoutBuf.String()), "\n")
		require.GreaterOrEqual(t, len(lines), 2)
		assert.Equal(t, []string{"Name", "Value"}, strings.Fields(lines[0]))
		assert.Equal(t, []string{"a", "1"}, strings.Fields(lines[len(lines)-1]))
	})

	t.Run("json", func(t *testing.T) {
		stdoutBuf, _ := interceptCmdOutput(t)

		f := mustNewFormatter(flag.FormatJSON, hdrs)
		mustWriteRow(f, "a", 1)
		require.NoError(t, f.Flush())

		var got []map[string]any
		require.NoError(t, json.Unmarshal(stdoutBuf.Bytes(), &got), stdoutBuf.String())
		assert.Equal(t, []map[string]any{{"Name": "a", "Value": float64(1)}}, got)
	})

	t.Run("unknown format is a bug", func(t *testing.T) {
		assert.PanicsWithValue(t,
			`BUG: newFormatter: unsupported formatter name: "xml"`,
			func() { mustNewFormatter("xml", hdrs) },
		)
	})
}

func TestUntrackedFilesExistErrMsg(t *testing.T) {
	msg := untrackedFilesExistErrMsg([]string{"a.txt", "dir/b.txt"})

	assert.Contains(t, msg, "--require-clean-git-worktree")
	assert.Contains(t, msg, "a.txt, dir/b.txt")

	many := make([]string, 0, 12)
	for i := range 12 {
		many = append(many, fmt.Sprintf("file%02d", i))
	}

	msg = untrackedFilesExistErrMsg(many)
	assert.Contains(t, msg, "file09")
	assert.NotContains(t, msg, "file10", "only the first 10 files are listed")
	assert.Contains(t, msg, "[...]")
}

func TestPSQLURLFromEnvOrConfig(t *testing.T) {
	initTest(t)

	repoCfg := &cfg.Repository{Database: cfg.Database{PGSQLURL: "postgres://from-config"}}

	t.Run("environment variable has precedence", func(t *testing.T) {
		t.Setenv(envVarPSQLURL, "postgres://from-env")

		assert.Equal(t, "postgres://from-env", getPSQLURIEnv())
		assert.Equal(t, "postgres://from-env", mustGetPSQLURI(repoCfg))

		url, err := postgresqlURL()
		require.NoError(t, err)
		assert.Equal(t, "postgres://from-env", url)
	})

	t.Run("config is used when the variable is empty", func(t *testing.T) {
		t.Setenv(envVarPSQLURL, "")

		assert.Empty(t, getPSQLURIEnv())
		assert.Equal(t, "postgres://from-config", mustGetPSQLURI(repoCfg))
	})

	t.Run("missing URL terminates the program", func(t *testing.T) {
		t.Setenv(envVarPSQLURL, "")
		_, stderrBuf := interceptCmdOutput(t)

		expectExit(t, exitCodeError, func() { mustGetPSQLURI(&cfg.Repository{}) })

		assert.Contains(t, stderrBuf.String(), "PostgreSQL connection information is missing")
	})
}

func TestPostgresqlURLFromRepositoryConfig(t *testing.T) {
	initTest(t)
	t.Setenv(envVarPSQLURL, "")

	t.Run("URL from repository config", func(t *testing.T) {
		r := repotest.CreateBaurRepository(t)
		setGlobal(t, &repositoryPath, r.Dir)

		url, err := postgresqlURL()
		require.NoError(t, err)
		assert.Equal(t, r.Cfg.Database.PGSQLURL, url)
	})

	t.Run("repository config without URL", func(t *testing.T) {
		r := repotest.CreateBaurRepository(t)
		r.Cfg.Database.PGSQLURL = ""
		require.NoError(t, r.Cfg.ToFile(filepath.Join(r.Dir, baur.RepositoryCfgFile), cfg.ToFileOptOverwrite()))
		setGlobal(t, &repositoryPath, r.Dir)

		_, err := postgresqlURL()
		require.ErrorIs(t, err, ErrPSQLURIMissing)
	})

	t.Run("directory without repository config", func(t *testing.T) {
		setGlobal(t, &repositoryPath, t.TempDir())

		_, err := postgresqlURL()
		require.ErrorContains(t, err, "baur repository not found")
	})
}

func TestFindRepository(t *testing.T) {
	initTest(t)
	r := repotest.CreateBaurRepository(t)

	t.Run("--repository flag", func(t *testing.T) {
		setGlobal(t, &repositoryPath, r.Dir)

		repo, err := findRepository()
		require.NoError(t, err)
		assert.Equal(t, r.Dir, repo.Path)
		assert.Equal(t, filepath.Join(r.Dir, baur.RepositoryCfgFile), repo.CfgPath)
	})

	t.Run("--repository flag with directory that is not a repository", func(t *testing.T) {
		dir := t.TempDir()
		setGlobal(t, &repositoryPath, dir)

		_, err := findRepository()
		require.ErrorContains(t, err, "baur repository not found")
		require.ErrorContains(t, err, filepath.Join(dir, baur.RepositoryCfgFile))
	})

	t.Run("searches parent directories of the working directory", func(t *testing.T) {
		setGlobal(t, &repositoryPath, "")
		sub := filepath.Join(r.Dir, "a", "b")
		fstest.MkdirAll(t, sub)
		t.Chdir(sub)

		repo, err := findRepository()
		require.NoError(t, err)
		assert.Equal(t, r.Dir, repo.Path)
	})

	t.Run("mustFindRepository outside of a repository terminates the program", func(t *testing.T) {
		setGlobal(t, &repositoryPath, "")
		t.Chdir(t.TempDir())
		_, stderrBuf := interceptCmdOutput(t)

		expectExit(t, exitCodeError, func() { mustFindRepository() })

		assert.Contains(t, stderrBuf.String(), "baur repository not found")
	})

	t.Run("mustFindRepository returns the repository", func(t *testing.T) {
		setGlobal(t, &repositoryPath, r.Dir)

		assert.Equal(t, r.Dir, mustFindRepository().Path)
	})
}

func TestMustGetRepoState(t *testing.T) {
	initTest(t)

	t.Run("git repository", func(t *testing.T) {
		r := newGitRepoWithApps(t)

		state := mustGetRepoState(r.Dir)

		assert.Equal(t, git.Name, state.Name())
		commit, err := state.CommitID()
		require.NoError(t, err)
		assert.Len(t, commit, 40)
	})

	t.Run("directory that is not part of a git repository", func(t *testing.T) {
		_, stderrBuf := interceptCmdOutput(t)

		expectExit(t, exitCodeError, func() { mustGetRepoState(t.TempDir()) })

		assert.Contains(t, stderrBuf.String(), "git repository not found")
	})
}

func TestMustUntrackedFilesNotExist(t *testing.T) {
	r := newGitRepoWithApps(t)
	state := mustGetRepoState(r.Dir)

	t.Run("flag disabled ignores untracked files", func(t *testing.T) {
		fstest.WriteToFile(t, []byte("x"), filepath.Join(r.Dir, "untracked.txt"))

		assert.NotPanics(t, func() { mustUntrackedFilesNotExist(false, state) })
	})

	t.Run("flag enabled with untracked file terminates the program", func(t *testing.T) {
		_, stderrBuf := interceptCmdOutput(t)

		expectExit(t, exitCodeError, func() { mustUntrackedFilesNotExist(true, mustGetRepoState(r.Dir)) })

		assert.Contains(t, stderrBuf.String(), "untracked.txt")
	})

	t.Run("flag enabled with clean worktree", func(t *testing.T) {
		require.NoError(t, os.Remove(filepath.Join(r.Dir, "untracked.txt")))

		assert.NotPanics(t, func() { mustUntrackedFilesNotExist(true, mustGetRepoState(r.Dir)) })
	})
}

func TestArgToTasksAndApps(t *testing.T) {
	r := newGitRepoWithApps(t)
	repo := mustFindRepository()
	state := mustGetRepoState(r.Dir)

	t.Run("mustArgToTasks", func(t *testing.T) {
		tasks := mustArgToTasks(repo, state, []string{"simpleApp.*"})

		ids := make([]string, 0, len(tasks))
		for _, task := range tasks {
			ids = append(ids, task.ID)
		}

		assert.ElementsMatch(t, []string{"simpleApp.build", "simpleApp.check"}, ids)
	})

	t.Run("mustArgToTask with a single match", func(t *testing.T) {
		assert.Equal(t, "simpleApp.build", mustArgToTask(repo, state, "simpleApp.build").ID)
	})

	t.Run("mustArgToTask matching multiple tasks terminates the program", func(t *testing.T) {
		_, stderrBuf := interceptCmdOutput(t)

		expectExit(t, exitCodeError, func() { mustArgToTask(repo, state, "simpleApp.*") })

		assert.Contains(t, stderrBuf.String(), "matches multiple tasks")
	})

	t.Run("mustArgToApp with a single match", func(t *testing.T) {
		assert.Equal(t, "simpleApp", mustArgToApp(repo, "simpleApp").Name)
	})

	t.Run("mustArgToApp matching multiple apps terminates the program", func(t *testing.T) {
		_, stderrBuf := interceptCmdOutput(t)

		expectExit(t, exitCodeError, func() { mustArgToApp(repo, "*") })

		assert.Contains(t, stderrBuf.String(), "matches multiple apps")
	})

	t.Run("mustArgToApps returns all apps", func(t *testing.T) {
		apps := mustArgToApps(repo, nil)
		names := make([]string, 0, len(apps))
		for _, app := range apps {
			names = append(names, app.Name)
		}

		assert.ElementsMatch(t, []string{"simpleApp", "appWithoutIO"}, names)
	})

	t.Run("unknown target terminates the program", func(t *testing.T) {
		expectExit(t, exitCodeError, func() { mustArgToTasks(repo, state, []string{"nope.build"}) })
		expectExit(t, exitCodeError, func() { mustArgToApps(repo, []string{"nope"}) })
	})

	t.Run("mustTaskRepoRelPath", func(t *testing.T) {
		task := mustArgToTask(repo, state, "simpleApp.build")

		assert.Equal(t, "simpleApp", mustTaskRepoRelPath(r.Dir, task))
	})
}

func TestMustArgToTasksFailsIfRepositoryHasNoTasks(t *testing.T) {
	initTest(t)
	r := repotest.CreateBaurRepository(t)
	r.CreateAppWithoutTasks(t)
	gittest.CommitFilesToGit(t, r.Dir)
	_, stderrBuf := interceptCmdOutput(t)

	expectExit(t, exitCodeError, func() {
		mustArgToTasks(mustFindRepository(), mustGetRepoState(r.Dir), nil)
	})

	assert.Contains(t, stderrBuf.String(), "could not find any tasks")
}

func TestArgToAppsFailsIfRepositoryHasNoApps(t *testing.T) {
	initTest(t)
	repotest.CreateBaurRepository(t)

	_, err := argToApps(mustFindRepository(), nil)

	require.ErrorContains(t, err, "could not find any applications")
}

type fakeOutput struct {
	baur.Output

	name      string
	exists    bool
	existsErr error
	size      uint64
	sizeErr   error
}

func (f *fakeOutput) Name() string               { return f.name }
func (f *fakeOutput) String() string             { return f.name }
func (f *fakeOutput) Exists() (bool, error)      { return f.exists, f.existsErr }
func (f *fakeOutput) SizeBytes() (uint64, error) { return f.size, f.sizeErr }

func TestDeclaredOutputsExist(t *testing.T) {
	initTest(t)
	task := &baur.Task{ID: "app.build"}

	tests := []struct {
		name       string
		outputs    []baur.Output
		want       bool
		wantStdout string
		wantStderr string
	}{
		{
			name:       "no outputs",
			want:       true,
			wantStdout: "does not produce outputs",
		},
		{
			name:       "existing output",
			outputs:    []baur.Output{&fakeOutput{name: "out.tar", exists: true, size: 2048}},
			want:       true,
			wantStdout: "created out.tar",
		},
		{
			name: "one of two outputs is missing",
			outputs: []baur.Output{
				&fakeOutput{name: "ok.tar", exists: true},
				&fakeOutput{name: "missing.tar"},
			},
			want:       false,
			wantStdout: "created ok.tar",
			wantStderr: "missing.tar as output defined but it was not created",
		},
		{
			name:       "existence check fails",
			outputs:    []baur.Output{&fakeOutput{name: "x", existsErr: errors.New("check-failed")}},
			want:       false,
			wantStderr: "check-failed",
		},
		{
			name:       "size query fails",
			outputs:    []baur.Output{&fakeOutput{name: "x", exists: true, sizeErr: errors.New("size-failed")}},
			want:       false,
			wantStderr: "size-failed",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stdoutBuf, stderrBuf := interceptCmdOutput(t)

			assert.Equal(t, tc.want, declaredOutputsExist(task, tc.outputs))

			assert.Contains(t, stdoutBuf.String(), tc.wantStdout)
			assert.Contains(t, stderrBuf.String(), tc.wantStderr)
		})
	}
}
