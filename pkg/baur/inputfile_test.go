package baur

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v2/internal/testutils/fstest"
)

func TestDigestDoesNotDependOnRepoPath(t *testing.T) {
	tempdir := t.TempDir()

	repoPath1 := filepath.Join(tempdir, "repo1")
	repoPath2 := filepath.Join(tempdir, "repo2")

	relFilepath1 := filepath.Join("appdir", "file1")
	relFilepath2 := filepath.Join("appdir", "file1")

	fstest.WriteToFile(t, []byte("hello"), filepath.Join(repoPath1, relFilepath1))
	fstest.WriteToFile(t, []byte("hello"), filepath.Join(repoPath2, relFilepath2))

	f1 := NewInputFile(repoPath1, relFilepath1)
	f2 := NewInputFile(repoPath2, relFilepath2)

	d1, err := f1.Digest()
	require.NoError(t, err)

	d2, err := f2.Digest()
	require.NoError(t, err)

	assert.Equal(t, d1.String(), d2.String())
}
