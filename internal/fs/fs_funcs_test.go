package fs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v5/internal/testutils/fstest"
)

func TestIsFileIsDirIsRegularFile(t *testing.T) {
	dir := fstest.TempDir(t)
	file := filepath.Join(dir, "f")
	fstest.WriteToFile(t, []byte("x"), file)
	link := filepath.Join(dir, "link")
	fstest.Symlink(t, file, link)

	isFile, err := IsFile(file)
	require.NoError(t, err)
	assert.True(t, isFile)

	isFile, err = IsFile(dir)
	require.NoError(t, err)
	assert.False(t, isFile)

	isDir, err := IsDir(dir)
	require.NoError(t, err)
	assert.True(t, isDir)

	isDir, err = IsDir(file)
	require.NoError(t, err)
	assert.False(t, isDir)

	reg, err := IsRegularFile(link)
	require.NoError(t, err, "symlinks are followed")
	assert.True(t, reg)

	for _, fn := range []func(string) (bool, error){IsFile, IsDir, IsRegularFile} {
		_, err = fn(filepath.Join(dir, "missing"))
		require.ErrorIs(t, err, os.ErrNotExist)
	}

	assert.True(t, FileExists(file))
	assert.False(t, FileExists(dir), "a directory is not a file")
	assert.False(t, FileExists(filepath.Join(dir, "missing")))
}

func TestDirsExist(t *testing.T) {
	dir := fstest.TempDir(t)
	file := filepath.Join(dir, "f")
	fstest.WriteToFile(t, nil, file)

	require.NoError(t, DirsExist())
	require.NoError(t, DirsExist(dir, dir))

	err := DirsExist(dir, filepath.Join(dir, "missing"))
	require.ErrorIs(t, err, os.ErrNotExist)
	assert.Contains(t, err.Error(), "missing")

	err = DirsExist(file)
	require.ErrorContains(t, err, "is not a directory")
}

func TestSameFile(t *testing.T) {
	dir := fstest.TempDir(t)
	a, b := filepath.Join(dir, "a"), filepath.Join(dir, "b")
	fstest.WriteToFile(t, []byte("x"), a)
	fstest.WriteToFile(t, []byte("x"), b)
	link := filepath.Join(dir, "link")
	fstest.Symlink(t, a, link)

	same, err := SameFile(a, link)
	require.NoError(t, err)
	assert.True(t, same)

	same, err = SameFile(a, b)
	require.NoError(t, err)
	assert.False(t, same, "equal content does not make them the same file")

	_, err = SameFile(a, filepath.Join(dir, "missing"))
	require.Error(t, err)
	_, err = SameFile(filepath.Join(dir, "missing"), a)
	require.Error(t, err)
}

func TestFindFileInParentDirs(t *testing.T) {
	dir := fstest.TempDir(t)
	fstest.WriteToFile(t, []byte("x"), filepath.Join(dir, "top.cfg"))
	fstest.MkdirAll(t, filepath.Join(dir, "a", "b"))

	got, err := FindFileInParentDirs(filepath.Join(dir, "a", "b"), "top.cfg")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "top.cfg"), got)

	_, err = FindFileInParentDirs(filepath.Join(dir, "a", "b"), "does-not-exist-1234")
	require.ErrorIs(t, err, os.ErrNotExist)
}

// An entry with the searched name but the wrong type is not a match.
func TestFindInParentDirsTypeMismatch(t *testing.T) {
	dir := fstest.TempDir(t)
	fstest.MkdirAll(t, filepath.Join(dir, "sub"))
	fstest.MkdirAll(t, filepath.Join(dir, "adir"))
	fstest.WriteToFile(t, nil, filepath.Join(dir, "afile"))
	start := filepath.Join(dir, "sub")

	t.Run("FindDirInParentDirs finds a directory", func(t *testing.T) {
		got, err := FindDirInParentDirs(start, "adir")
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(dir, "adir"), got)
	})

	t.Run("FindDirInParentDirs does not match a regular file", func(t *testing.T) {
		got, err := FindDirInParentDirs(start, "afile")
		require.ErrorIs(t, err, os.ErrNotExist)
		assert.Empty(t, got)
	})

	t.Run("FindFileInParentDirs does not match a directory", func(t *testing.T) {
		got, err := FindFileInParentDirs(start, "adir")
		require.ErrorIs(t, err, os.ErrNotExist)
		assert.Empty(t, got)
	})

	t.Run("a wrong type entry does not stop the search in the parent directories", func(t *testing.T) {
		fstest.MkdirAll(t, filepath.Join(dir, "sub", "name"))
		fstest.WriteToFile(t, nil, filepath.Join(dir, "name"))

		got, err := FindFileInParentDirs(start, "name")

		require.NoError(t, err)
		assert.Equal(t, filepath.Join(dir, "name"), got)
	})
}

func TestFindFilesInSubDir(t *testing.T) {
	dir := fstest.TempDir(t)
	for _, f := range []string{"t.cfg", "a/t.cfg", "a/b/t.cfg", "a/b/c/t.cfg", "other.cfg"} {
		fstest.WriteToFile(t, nil, filepath.Join(dir, f))
	}

	tests := []struct {
		maxdepth int
		want     []string
	}{
		{0, []string{"t.cfg"}},
		{1, []string{"t.cfg", "a/t.cfg"}},
		{2, []string{"t.cfg", "a/t.cfg", "a/b/t.cfg"}},
		{10, []string{"t.cfg", "a/t.cfg", "a/b/t.cfg", "a/b/c/t.cfg"}},
	}

	for _, tc := range tests {
		got, err := FindFilesInSubDir(dir, "t.cfg", tc.maxdepth)
		require.NoError(t, err)

		want := make([]string, 0, len(tc.want))
		for _, w := range tc.want {
			want = append(want, filepath.Join(dir, filepath.FromSlash(w)))
		}

		assert.Equal(t, want, got, "maxdepth %d: shallower matches come first", tc.maxdepth)
	}

	got, err := FindFilesInSubDir(dir, "[", 1)
	require.Error(t, err)
	assert.Nil(t, got)
}

func TestPathsJoinAndAbsPaths(t *testing.T) {
	root := filepath.FromSlash("/root")

	assert.Equal(t,
		[]string{filepath.FromSlash("/root/a/b"), filepath.FromSlash("/root/c")},
		PathsJoin(root, []string{"a/./b", "x/../c"}),
	)
	assert.Empty(t, PathsJoin(root, nil))

	assert.Equal(t,
		[]string{filepath.FromSlash("/root/a"), filepath.FromSlash("/abs")},
		AbsPaths(root, []string{"a", filepath.FromSlash("/abs")}),
	)

	assert.Equal(t, filepath.FromSlash("/abs"), AbsPath(root, filepath.FromSlash("/abs")))
	assert.Equal(t, filepath.FromSlash("/root/rel"), AbsPath(root, "rel"))
}

func TestFileReadLine(t *testing.T) {
	dir := fstest.TempDir(t)

	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"first line keeps newline", "first\nsecond\n", "first\n"},
		{"no trailing newline", "only", "only"},
		{"empty file", "", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := filepath.Join(dir, tc.name)
			fstest.WriteToFile(t, []byte(tc.content), p)

			got, err := FileReadLine(p)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}

	_, err := FileReadLine(filepath.Join(dir, "missing"))
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestFileSize(t *testing.T) {
	dir := fstest.TempDir(t)
	p := filepath.Join(dir, "f")
	fstest.WriteToFile(t, []byte("12345"), p)

	size, err := FileSize(p)
	require.NoError(t, err)
	assert.EqualValues(t, 5, size)

	size, err = FileSize(filepath.Join(dir, "missing"))
	require.Error(t, err)
	assert.EqualValues(t, -1, size)
}

func TestMkdirCreatesNestedDirsAndIsIdempotent(t *testing.T) {
	p := filepath.Join(fstest.TempDir(t), "a", "b", "c")

	require.NoError(t, Mkdir(p))
	require.NoError(t, Mkdir(p))
	assert.DirExists(t, p)
}

func TestBackupFile(t *testing.T) {
	dir := fstest.TempDir(t)
	p := filepath.Join(dir, "f.toml")
	fstest.WriteToFile(t, []byte("content"), p)

	require.NoError(t, BackupFile(p))

	assert.NoFileExists(t, p)
	assert.Equal(t, "content", string(fstest.ReadFile(t, p+FileBackupSuffix)))
	require.Error(t, BackupFile(p), "backup of a missing file fails")
}

func TestRealPathAndRealPathRel(t *testing.T) {
	dir := fstest.TempDir(t)
	fstest.WriteToFile(t, nil, filepath.Join(dir, "real", "f"))
	fstest.Symlink(t, filepath.Join(dir, "real"), filepath.Join(dir, "link"))

	got, err := RealPath(filepath.Join(dir, "link", "f"))
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "real", "f"), got)

	rel, err := RealPathRel(dir, filepath.Join(dir, "link", "f"))
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("real", "f"), rel)

	_, err = RealPath(filepath.Join(dir, "missing"))
	require.ErrorIs(t, err, os.ErrNotExist)

	_, err = RealPathRel(dir, filepath.Join(dir, "missing"))
	require.Error(t, err)
}

func TestOwnerHasExecPerm(t *testing.T) {
	assert.True(t, OwnerHasExecPerm(0o700))
	assert.True(t, OwnerHasExecPerm(0o100))
	assert.False(t, OwnerHasExecPerm(0o677), "only the owner bit counts")
	assert.False(t, OwnerHasExecPerm(0o011))

	dir := fstest.TempDir(t)
	exe, plain := filepath.Join(dir, "exe"), filepath.Join(dir, "plain")
	fstest.WriteExecutable(t, nil, exe)
	fstest.WriteToFile(t, nil, plain)

	got, err := FileHasOwnerExecPerm(exe)
	require.NoError(t, err)
	assert.True(t, got)

	got, err = FileHasOwnerExecPerm(plain)
	require.NoError(t, err)
	assert.False(t, got)

	_, err = FileHasOwnerExecPerm(filepath.Join(dir, "missing"))
	require.Error(t, err)
}
