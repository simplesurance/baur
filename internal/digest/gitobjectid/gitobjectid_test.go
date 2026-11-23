package gitobjectid

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v5/internal/digest"
	"github.com/simplesurance/baur/v5/internal/testutils/fstest"
	"github.com/simplesurance/baur/v5/internal/testutils/gittest"
)

func TestFileReturnsGitObjectID(t *testing.T) {
	dir := fstest.TempDir(t)
	gittest.CreateRepository(t, dir)

	f := filepath.Join(dir, "hello.txt")
	fstest.WriteToFile(t, []byte("hello\n"), f)

	var logged int
	h := New(dir, func(string, ...any) { logged++ })

	d, err := h.File(f)
	require.NoError(t, err)

	// output of: printf 'hello\n' | git hash-object --stdin
	assert.Equal(t, "gitobjectid:ce013625030ba8dba906f756967f9e9ca394464a", d.String())
	assert.Equal(t, digest.GitObjectID, d.Algorithm)
	assert.Equal(t, 1, logged)
}

func TestFileDifferentContentDifferentID(t *testing.T) {
	dir := fstest.TempDir(t)
	gittest.CreateRepository(t, dir)

	a, b := filepath.Join(dir, "a"), filepath.Join(dir, "b")
	fstest.WriteToFile(t, []byte("1"), a)
	fstest.WriteToFile(t, []byte("2"), b)

	h := New(dir, func(string, ...any) {})
	da, err := h.File(a)
	require.NoError(t, err)
	db, err := h.File(b)
	require.NoError(t, err)

	assert.NotEqual(t, da.String(), db.String())
}

func TestFileMissingFileIsAnError(t *testing.T) {
	dir := fstest.TempDir(t)
	gittest.CreateRepository(t, dir)

	_, err := New(dir, func(string, ...any) {}).File(filepath.Join(dir, "missing"))

	require.Error(t, err)
}
