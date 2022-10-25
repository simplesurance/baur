package fs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/simplesurance/baur/v3/internal/testutils/fstest"
)

func TestFindFileInParentDirsOnRoot(t *testing.T) {
	_, err := FindFileInParentDirs(filepath.FromSlash("/"), "mytestfile-which-must-not-exist-1234")
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestFindFileInParentDirWithExcessivePathSeperator(t *testing.T) {
	var err error
	tempdir := fstest.TempDir(t)

	const wantedFilename = ".baur.cfg"
	const subdir1 = "subdir1"
	subdir2AbsPath := filepath.Join(tempdir, subdir1, "subdir2")
	wantedFileAbsPath := filepath.Join(tempdir, subdir1, wantedFilename)

	fstest.WriteToFile(t, []byte("hello"), filepath.Join(tempdir, subdir1, wantedFilename))

	foundPath, err := FindFileInParentDirs(subdir2AbsPath+string(os.PathSeparator), wantedFilename)
	assert.NoError(t, err)
	assert.Equal(t, wantedFileAbsPath, foundPath)
}
