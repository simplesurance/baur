package git

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v5/internal/exec"
	"github.com/simplesurance/baur/v5/internal/testutils/fstest"
	"github.com/simplesurance/baur/v5/internal/testutils/gittest"
)

var commitIDRegex = regexp.MustCompile(`^[0-9a-f]{40}$`)

// newCommittedRepo creates a git repository with the given tracked files, committed.
func newCommittedRepo(t *testing.T, files map[string]string) string {
	t.Helper()

	dir := fstest.TempDir(t)
	gittest.CreateRepository(t, dir)

	for name, content := range files {
		fstest.WriteToFile(t, []byte(content), filepath.Join(dir, name))
	}

	gittest.CommitFilesToGit(t, dir)

	return dir
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()

	_, err := exec.Command("git", args...).Directory(dir).ExpectSuccess().Run(t.Context())
	require.NoError(t, err)
}

func TestCommandIsInstalled(t *testing.T) {
	assert.True(t, CommandIsInstalled())

	t.Setenv("PATH", t.TempDir())
	assert.False(t, CommandIsInstalled())
}

func TestIsGitDir(t *testing.T) {
	dir := newCommittedRepo(t, map[string]string{"sub/f": "x"})

	t.Run("repository root", func(t *testing.T) {
		ok, err := IsGitDir(dir)
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("subdirectory of a repository", func(t *testing.T) {
		ok, err := IsGitDir(filepath.Join(dir, "sub"))
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("directory outside of a repository", func(t *testing.T) {
		ok, err := IsGitDir(fstest.TempDir(t))
		require.NoError(t, err)
		assert.False(t, ok)
	})
}

func TestCommitID(t *testing.T) {
	dir := newCommittedRepo(t, map[string]string{"f": "1"})

	id, err := CommitID(dir)
	require.NoError(t, err)
	assert.Regexp(t, commitIDRegex, id)

	fstest.WriteToFile(t, []byte("2"), filepath.Join(dir, "f"))
	runGit(t, dir, "commit", "-a", "-m", "second")

	id2, err := CommitID(dir)
	require.NoError(t, err)
	assert.Regexp(t, commitIDRegex, id2)
	assert.NotEqual(t, id, id2, "a new commit must result in a new commit id")
}

func TestCommitIDOutsideOfRepositoryIsAnError(t *testing.T) {
	_, err := CommitID(fstest.TempDir(t))

	require.Error(t, err)
}

// git writes to stdout and stderr at the same time if HEAD does not exist.
func TestCommitIDInRepositoryWithoutCommitsIsAnError(t *testing.T) {
	dir := fstest.TempDir(t)
	gittest.CreateRepository(t, dir)

	_, err := CommitID(dir)

	require.Error(t, err)
}

func TestWorktreeIsDirty(t *testing.T) {
	tests := []struct {
		name      string
		modify    func(t *testing.T, dir string)
		wantDirty bool
	}{
		{"clean", func(*testing.T, string) {}, false},
		{"untracked file", func(t *testing.T, dir string) {
			fstest.WriteToFile(t, []byte("x"), filepath.Join(dir, "new"))
		}, true},
		{"modified tracked file", func(t *testing.T, dir string) {
			fstest.WriteToFile(t, []byte("changed"), filepath.Join(dir, "tracked"))
		}, true},
		{"staged change", func(t *testing.T, dir string) {
			fstest.WriteToFile(t, []byte("changed"), filepath.Join(dir, "tracked"))
			runGit(t, dir, "add", "tracked")
		}, true},
		{"deleted tracked file", func(t *testing.T, dir string) {
			require.NoError(t, os.Remove(filepath.Join(dir, "tracked")))
		}, true},
		{"renamed tracked file", func(t *testing.T, dir string) {
			runGit(t, dir, "mv", "tracked", "renamed")
		}, true},
		{"only a gitignored file", func(t *testing.T, dir string) {
			fstest.WriteToFile(t, []byte("ignored"), filepath.Join(dir, "ignored.log"))
		}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := newCommittedRepo(t, map[string]string{"tracked": "orig", ".gitignore": "*.log\n"})
			tc.modify(t, dir)

			dirty, err := WorktreeIsDirty(dir)

			require.NoError(t, err)
			assert.Equal(t, tc.wantDirty, dirty)
		})
	}
}

func TestWorktreeIsDirtyOutsideOfRepositoryIsAnError(t *testing.T) {
	_, err := WorktreeIsDirty(fstest.TempDir(t))

	require.Error(t, err)
}

func TestUntrackedFilesIgnoresTrackedChangesAndRenames(t *testing.T) {
	dir := newCommittedRepo(t, map[string]string{"tracked": "orig", "other": "o"})

	fstest.WriteToFile(t, []byte("changed"), filepath.Join(dir, "tracked"))
	runGit(t, dir, "mv", "other", "renamed")
	fstest.WriteToFile(t, []byte("n"), filepath.Join(dir, "new"))

	got, err := UntrackedFiles(dir)

	require.NoError(t, err)
	assert.Equal(t, []string{"new"}, got)
}

func TestUntrackedFilesOfCleanRepoIsEmpty(t *testing.T) {
	dir := newCommittedRepo(t, map[string]string{"tracked": "x"})

	got, err := UntrackedFiles(dir)

	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestUntrackedFilesOutsideOfRepositoryIsAnError(t *testing.T) {
	_, err := UntrackedFiles(fstest.TempDir(t))

	require.Error(t, err)
}

// Git quotes special file names in the output of git status, unless -z is used.
// The returned paths must be the plain file names, WithoutUntracked compares
// them with the paths of resolved inputs.
func TestUntrackedFilesReturnsPlainPathsForSpecialFilenames(t *testing.T) {
	dir := newCommittedRepo(t, map[string]string{"tracked": "x"})

	names := []string{
		"with space.txt",
		"plain.txt",
		"ümlaut.txt",
		`with"quote.txt`,
		"sub dir/in sub dir.txt",
		"-leading-dash.txt",
	}
	for _, n := range names {
		fstest.WriteToFile(t, []byte("1"), filepath.Join(dir, n))
	}

	got, err := UntrackedFiles(dir)

	require.NoError(t, err)
	assert.ElementsMatch(t, names, got)
}

func TestWithoutUntrackedExcludesSpecialFilenames(t *testing.T) {
	dir := newCommittedRepo(t, map[string]string{"tracked file.txt": "x"})
	fstest.WriteToFile(t, []byte("1"), filepath.Join(dir, "untracked file.txt"))
	fstest.WriteToFile(t, []byte("1"), filepath.Join(dir, "ümlaut.txt"))

	got, err := NewRepository(dir).WithoutUntracked("tracked file.txt", "untracked file.txt", "ümlaut.txt")

	require.NoError(t, err)
	assert.Equal(t, []string{"tracked file.txt"}, got)
}

// In the output of git status -z a rename entry is followed by a second record
// containing the original path. It must not be interpreted as a status entry.
// The original path looks like an untracked entry, to detect if it is parsed.
func TestUntrackedFilesIgnoresRenameOriginPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("'?' is not allowed in windows file names")
	}

	dir := newCommittedRepo(t, map[string]string{"?? fake.txt": "content"})

	_, err := exec.Command("git", "mv", "?? fake.txt", "new name.txt").Directory(dir).ExpectSuccess().Run(t.Context())
	require.NoError(t, err)
	fstest.WriteToFile(t, []byte("1"), filepath.Join(dir, "untracked.txt"))

	got, err := UntrackedFiles(dir)

	require.NoError(t, err)
	assert.Equal(t, []string{"untracked.txt"}, got)
}

func TestObjectID(t *testing.T) {
	dir := fstest.TempDir(t)
	gittest.CreateRepository(t, dir)
	f := filepath.Join(dir, "hello.txt")
	fstest.WriteToFile(t, []byte("hello\n"), f)

	id, err := ObjectID(t.Context(), f, "hello.txt")
	require.NoError(t, err)
	assert.Equal(t, "ce013625030ba8dba906f756967f9e9ca394464a", id)

	_, err = ObjectID(t.Context(), filepath.Join(dir, "missing"), "missing")
	require.Error(t, err)
}

func TestNewRepositoryWithCheck(t *testing.T) {
	t.Run("git repository", func(t *testing.T) {
		dir := newCommittedRepo(t, map[string]string{"f": "x"})

		r, err := NewRepositoryWithCheck(dir)

		require.NoError(t, err)
		assert.Equal(t, "git", r.Name())
	})

	t.Run("not a git repository", func(t *testing.T) {
		_, err := NewRepositoryWithCheck(fstest.TempDir(t))

		require.ErrorIs(t, err, ErrRepositoryNotFound)
	})

	t.Run("git command is missing", func(t *testing.T) {
		dir := newCommittedRepo(t, map[string]string{"f": "x"})
		t.Setenv("PATH", t.TempDir())

		_, err := NewRepositoryWithCheck(dir)

		require.ErrorContains(t, err, "git command not found")
		require.NotErrorIs(t, err, ErrRepositoryNotFound)
	})
}

func TestRepositoryCommitIDIsCached(t *testing.T) {
	dir := newCommittedRepo(t, map[string]string{"f": "1"})
	r := NewRepository(dir)

	first, err := r.CommitID()
	require.NoError(t, err)
	assert.Regexp(t, commitIDRegex, first)

	fstest.WriteToFile(t, []byte("2"), filepath.Join(dir, "f"))
	runGit(t, dir, "commit", "-a", "-m", "second")

	second, err := r.CommitID()
	require.NoError(t, err)
	assert.Equal(t, first, second, "the commit id is determined once per Repository")

	fresh, err := NewRepository(dir).CommitID()
	require.NoError(t, err)
	assert.NotEqual(t, first, fresh)
}

func TestRepositoryCommitIDErrorIsNotCached(t *testing.T) {
	dir := fstest.TempDir(t)
	r := NewRepository(dir)

	_, err := r.CommitID()
	require.Error(t, err, "dir is not a repository yet")

	gittest.CreateRepository(t, dir)
	fstest.WriteToFile(t, []byte("1"), filepath.Join(dir, "f"))
	gittest.CommitFilesToGit(t, dir)

	id, err := r.CommitID()
	require.NoError(t, err)
	assert.Regexp(t, commitIDRegex, id)
}

func TestRepositoryWorktreeIsDirtyIsCached(t *testing.T) {
	dir := newCommittedRepo(t, map[string]string{"f": "1"})
	r := NewRepository(dir)

	dirty, err := r.WorktreeIsDirty()
	require.NoError(t, err)
	assert.False(t, dirty)

	fstest.WriteToFile(t, []byte("new"), filepath.Join(dir, "untracked"))

	dirty, err = r.WorktreeIsDirty()
	require.NoError(t, err)
	assert.False(t, dirty, "the state is determined once per Repository")

	dirty, err = NewRepository(dir).WorktreeIsDirty()
	require.NoError(t, err)
	assert.True(t, dirty)
}

func TestRepositoryWorktreeIsDirtyError(t *testing.T) {
	_, err := NewRepository(fstest.TempDir(t)).WorktreeIsDirty()

	require.Error(t, err)
}

func TestRepositoryUntrackedFilesIsNotCached(t *testing.T) {
	dir := newCommittedRepo(t, map[string]string{"f": "1"})
	r := NewRepository(dir)

	got, err := r.UntrackedFiles()
	require.NoError(t, err)
	assert.Empty(t, got)

	fstest.WriteToFile(t, []byte("n"), filepath.Join(dir, "new"))

	got, err = r.UntrackedFiles()
	require.NoError(t, err)
	assert.Equal(t, []string{"new"}, got)
}

func TestWithoutUntracked(t *testing.T) {
	dir := newCommittedRepo(t, map[string]string{
		"tracked":        "1",
		"sub/tracked":    "1",
		".gitignore":     "*.log\n",
		"sub/.gitignore": "",
	})
	fstest.WriteToFile(t, []byte("n"), filepath.Join(dir, "untracked"))
	fstest.WriteToFile(t, []byte("n"), filepath.Join(dir, "sub", "untracked"))
	fstest.WriteToFile(t, []byte("n"), filepath.Join(dir, "build.log"))

	r := NewRepository(dir)

	t.Run("relative paths", func(t *testing.T) {
		got, err := r.WithoutUntracked(
			"tracked", "untracked", filepath.Join("sub", "tracked"), filepath.Join("sub", "untracked"),
		)

		require.NoError(t, err)
		assert.Equal(t, []string{"tracked", filepath.Join("sub", "tracked")}, got, "order is kept")
	})

	t.Run("absolute paths are returned unchanged", func(t *testing.T) {
		tracked := filepath.Join(dir, "tracked")

		got, err := r.WithoutUntracked(tracked, filepath.Join(dir, "untracked"))

		require.NoError(t, err)
		assert.Equal(t, []string{tracked}, got)
	})

	t.Run("gitignored files are removed", func(t *testing.T) {
		got, err := r.WithoutUntracked("build.log", "tracked")

		require.NoError(t, err)
		assert.Equal(t, []string{"tracked"}, got)
	})

	t.Run("unknown paths are kept", func(t *testing.T) {
		got, err := r.WithoutUntracked("does-not-exist")

		require.NoError(t, err)
		assert.Equal(t, []string{"does-not-exist"}, got)
	})

	t.Run("no paths", func(t *testing.T) {
		got, err := r.WithoutUntracked()

		require.NoError(t, err)
		assert.Empty(t, got)
	})
}

func TestWithoutUntrackedUsesSnapshotOfFirstCall(t *testing.T) {
	dir := newCommittedRepo(t, map[string]string{"f": "1"})
	r := NewRepository(dir)

	_, err := r.WithoutUntracked("f")
	require.NoError(t, err)

	fstest.WriteToFile(t, []byte("n"), filepath.Join(dir, "late"))

	got, err := r.WithoutUntracked("late")
	require.NoError(t, err)
	assert.Equal(t, []string{"late"}, got, "files created after the first call are not detected as untracked")
}

func TestWithoutUntrackedOutsideOfRepositoryIsAnError(t *testing.T) {
	_, err := NewRepository(fstest.TempDir(t)).WithoutUntracked("f")

	require.Error(t, err)
}

// IsGitDir must accept both layouts: ".git" is a directory in a normal
// repository and a file in git worktrees and submodules.
func TestIsGitDirLayouts(t *testing.T) {
	t.Run("normal repository", func(t *testing.T) {
		dir := newCommittedRepo(t, map[string]string{"f": "x"})
		fstest.MkdirAll(t, filepath.Join(dir, "sub", "deeper"))

		for _, d := range []string{dir, filepath.Join(dir, "sub", "deeper")} {
			ok, err := IsGitDir(d)
			require.NoError(t, err)
			assert.True(t, ok, d)
		}
	})

	t.Run("worktree or submodule with a .git file", func(t *testing.T) {
		dir := fstest.TempDir(t)
		fstest.WriteToFile(t, []byte("gitdir: /somewhere/else\n"), filepath.Join(dir, ".git"))
		fstest.MkdirAll(t, filepath.Join(dir, "sub"))

		for _, d := range []string{dir, filepath.Join(dir, "sub")} {
			ok, err := IsGitDir(d)
			require.NoError(t, err)
			assert.True(t, ok, d)
		}
	})

	t.Run("no repository", func(t *testing.T) {
		ok, err := IsGitDir(fstest.TempDir(t))

		require.NoError(t, err)
		assert.False(t, ok)
	})
}
