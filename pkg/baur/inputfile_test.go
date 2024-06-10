package baur

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v4/internal/digest/sha384"
	"github.com/simplesurance/baur/v4/internal/testutils/fstest"
)

func TestDigestDoesNotDependOnRepoPath(t *testing.T) {
	tempdir := t.TempDir()

	repoAbsPath1 := filepath.Join(tempdir, "repo1")
	repoAbsPath2 := filepath.Join(tempdir, "repo2")

	relFilepath1 := filepath.Join("appdir", "file1")
	relFilepath2 := filepath.Join("appdir", "file1")

	absFilepath1 := filepath.Join(repoAbsPath1, relFilepath1)
	absFilepath2 := filepath.Join(repoAbsPath2, relFilepath2)

	fstest.WriteToFile(t, []byte("hello"), absFilepath1)
	fstest.WriteToFile(t, []byte("hello"), absFilepath2)

	f1 := NewInputFile(absFilepath1, relFilepath1, false, WithHashFn(sha384.File))
	f2 := NewInputFile(absFilepath2, relFilepath2, false, WithHashFn(sha384.File))

	d1, err := f1.Digest()
	require.NoError(t, err)

	d2, err := f2.Digest()
	require.NoError(t, err)

	assert.Equal(t, d1.String(), d2.String())
}
