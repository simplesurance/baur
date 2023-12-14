package gitobjectid

import (
	"path/filepath"
	"testing"

	"github.com/simplesurance/baur/v3/internal/testutils/fstest"
	"github.com/simplesurance/baur/v3/internal/testutils/gittest"

	"github.com/stretchr/testify/require"
)

func TestCalculatedUntrackedAndReadTrackedFileIDsAreSame(t *testing.T) {
	const fRel = "file"
	tempDir := t.TempDir()

	gittest.CreateRepository(t, tempDir)

	f := filepath.Join(tempDir, fRel)
	fstest.WriteToFile(t, []byte("110"), f)

	calc := New(tempDir, t.Logf)
	dUntracked, err := calc.File(f)
	require.NoError(t, err)
	require.Empty(t, calc.objectIDs)
	require.Empty(t, calc.symlinkPaths)

	gittest.CommitFilesToGit(t, tempDir)
	calc = New(tempDir, t.Logf)
	dTracked, err := calc.File(f)
	require.NoError(t, err)
	require.Len(t, calc.objectIDs, 1)
	require.Empty(t, calc.symlinkPaths)

	require.Equal(t, dUntracked.String(), dTracked.String())

}

func TestObjectIDsOfModifiedFilesAreNotUsed(t *testing.T) {
	const fRel = "file"
	tempDir := t.TempDir()

	gittest.CreateRepository(t, tempDir)

	f := filepath.Join(tempDir, fRel)
	fstest.WriteToFile(t, []byte("110"), f)
	gittest.CommitFilesToGit(t, tempDir)

	calc := New(tempDir, t.Logf)
	oldD, err := calc.File(f)
	require.NoError(t, err)

	fstest.WriteToFile(t, []byte("112"), f)

	calc = New(tempDir, t.Logf)
	newD, err := calc.File(f)
	require.NoError(t, err)
	require.Empty(t, calc.objectIDs)
	require.Empty(t, calc.symlinkPaths)

	require.NotEqual(t, oldD, newD)
}
