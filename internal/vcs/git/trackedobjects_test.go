package git

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v5/internal/testutils/fstest"
	"github.com/simplesurance/baur/v5/internal/testutils/gittest"
)

func TestModifiedFilesMissing(t *testing.T) {
	const fRel = "file"
	ctx := t.Context()
	tempDir := t.TempDir()

	gittest.CreateRepository(t, tempDir)

	f := filepath.Join(tempDir, fRel)
	fstest.WriteToFile(t, []byte("110"), f)
	gittest.CommitFilesToGit(t, tempDir)

	calc := NewTrackedObjects(tempDir, t.Logf)
	_, err := calc.Get(ctx, f)
	require.NoError(t, err)

	fstest.WriteToFile(t, []byte("112"), f)

	calc = NewTrackedObjects(tempDir, t.Logf)
	_, err = calc.Get(ctx, f)
	require.ErrorIs(t, err, ErrObjectNotFound)
}

func TestCalculatedUntrackedAndReadTrackedFileIDsAreSame(t *testing.T) {
	const fRel = "file"
	tempDir := t.TempDir()

	gittest.CreateRepository(t, tempDir)

	f := filepath.Join(tempDir, fRel)
	fstest.WriteToFile(t, []byte("110"), f)

	idUntracked, err := ObjectID(t.Context(), f, fRel)
	require.NoError(t, err)

	gittest.CommitFilesToGit(t, tempDir)

	calc := NewTrackedObjects(tempDir, t.Logf)
	to, err := calc.Get(t.Context(), f)
	require.NoError(t, err)
	require.NotNil(t, to)

	require.Equal(t, to.ObjectID, idUntracked)
	assert.Equal(t, ObjectTypeFile, to.Mode&ObjectTypeFile)
}
