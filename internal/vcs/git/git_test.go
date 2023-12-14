package git

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v3/internal/exec"
	"github.com/simplesurance/baur/v3/internal/log"
	"github.com/simplesurance/baur/v3/internal/testutils/fstest"
	"github.com/simplesurance/baur/v3/internal/testutils/gittest"
)

func TestUntrackedFilesIncludesGitIgnoredFiles(t *testing.T) {
	const ignoredFileName = "f2"

	log.RedirectToTestingLog(t)
	oldExecDebugFfN := exec.DefaultLogFn
	exec.DefaultLogFn = t.Logf
	t.Cleanup(func() {
		exec.DefaultLogFn = oldExecDebugFfN
	})

	tempDir := t.TempDir()

	gittest.CreateRepository(t, tempDir)

	fstest.WriteToFile(t, []byte("abc"), filepath.Join(tempDir, ignoredFileName))

	untrackedFiles, err := UntrackedFiles(tempDir)
	require.NoError(t, err)
	require.Containsf(
		t,
		untrackedFiles,
		ignoredFileName,
		"UntrackedFiles() result is missing file %q which is in gitignore file",
		ignoredFileName,
	)
}

func TestUntrackedFilesIncludesFilesInSubdirs(t *testing.T) {
	const ignoredDirName = "ignored_dir"
	ignoredInSubdirFileName := filepath.Join(ignoredDirName, "f3")
	untrackedFilepathInSubdir := filepath.Join("a", "b", "c", "f4")

	log.RedirectToTestingLog(t)
	oldExecDebugFfN := exec.DefaultLogFn
	exec.DefaultLogFn = t.Logf
	t.Cleanup(func() {
		exec.DefaultLogFn = oldExecDebugFfN
	})

	tempDir := t.TempDir()

	gittest.CreateRepository(t, tempDir)

	fstest.WriteToFile(t, []byte("xyz"), filepath.Join(tempDir, ignoredInSubdirFileName))
	fstest.WriteToFile(t, []byte(fmt.Sprintf("%s%c\n", ignoredDirName, filepath.Separator)),
		filepath.Join(tempDir, ".gitignore"),
	)

	fstest.WriteToFile(t, []byte("xyyy"), filepath.Join(tempDir, untrackedFilepathInSubdir))

	untrackedFiles, err := UntrackedFiles(tempDir)
	require.NoError(t, err)

	require.Containsf(
		t,
		untrackedFiles,
		untrackedFilepathInSubdir,
		"UntrackedFiles() result is missing file %q in subdir", untrackedFilepathInSubdir,
	)

	require.Containsf(
		t,
		untrackedFiles,
		ignoredInSubdirFileName,
		"UntrackedFiles() result is missing file %q in subdir which is in gitignore file", untrackedFilepathInSubdir,
	)
}

func TestUntrackedFilesDoesNotContainTrackedFile(t *testing.T) {
	const trackedFilename = "f1"

	log.RedirectToTestingLog(t)
	oldExecDebugFfN := exec.DefaultLogFn
	exec.DefaultLogFn = t.Logf
	t.Cleanup(func() {
		exec.DefaultLogFn = oldExecDebugFfN
	})

	tempDir := t.TempDir()

	gittest.CreateRepository(t, tempDir)

	fstest.WriteToFile(t, []byte("abc"), filepath.Join(tempDir, trackedFilename))
	gittest.CommitFilesToGit(t, tempDir)

	untrackedFiles, err := UntrackedFiles(tempDir)
	require.NoError(t, err)

	require.Empty(t, untrackedFiles)
}
