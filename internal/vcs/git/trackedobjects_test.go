package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/simplesurance/baur/v3/internal/testutils/fstest"
	"github.com/simplesurance/baur/v3/internal/testutils/gittest"
	"github.com/stretchr/testify/require"
)

func TestModifiedFilesMissing(t *testing.T) {
	const fRel = "file"
	ctx := context.Background()
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
	require.ErrorIs(t, err, os.ErrNotExist)
}
