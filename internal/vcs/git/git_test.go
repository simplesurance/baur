package git

// The inputresolver_test.go file contains further testcases that test the
// functionality of the package.

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v1/internal/testutils/fstest"
	"github.com/simplesurance/baur/v1/internal/testutils/gittest"
)

func TestSplitArgsMaxBiggerThenFirstElem(t *testing.T) {
	testArgs := []string{"/etc/", "/tmp/"}

	spl, remaining := splitArgs(testArgs, 2)
	assert.EqualValues(t, []string{"/etc/"}, spl)
	assert.EqualValues(t, []string{"/tmp/"}, remaining)

	spl, remaining = splitArgs(remaining, 2)
	assert.EqualValues(t, []string{"/tmp/"}, spl)
	assert.Empty(t, remaining)

}

func TestSplitArgsMaxBiggerThenAll(t *testing.T) {
	testArgs := []string{"/etc/", "/tmp/"}

	spl, remaining := splitArgs(testArgs, 100)
	assert.EqualValues(t, []string{"/etc/", "/tmp/"}, spl)
	assert.Empty(t, remaining)
}

func TestSplitArgsMaxExact(t *testing.T) {
	testArgs := []string{"/etc/", "/tmp/"}

	spl, remaining := splitArgs(testArgs, 10)
	assert.EqualValues(t, []string{"/etc/", "/tmp/"}, spl)
	assert.Empty(t, remaining)
}

func TestLsFiles(t *testing.T) {
	tempDir := t.TempDir()
	gittest.CreateRepository(t, tempDir)

	fname1 := "hello.txt"
	fname2 := "bye.txt"

	fstest.WriteToFile(t, []byte("1"), filepath.Join(tempDir, fname1))
	fstest.WriteToFile(t, []byte("2"), filepath.Join(tempDir, fname2))
	gittest.CommitFilesToGit(t, tempDir)

	for i := 1; i < len(fname1)+len(fname2)+2; i++ {
		t.Run(fmt.Sprintf("maxArgs%d", i), func(t *testing.T) {
			res, err := lsFilesArgSpl(1, tempDir, []string{fname1, fname2})
			require.NoError(t, err)
			require.ElementsMatch(
				t,
				[]string{fname1, fname2},
				res,
			)
		})
	}
}

func TestLsFilesNoOutputResolvesToNoPaths(t *testing.T) {
	tempDir := t.TempDir()
	gittest.CreateRepository(t, tempDir)

	fname1 := "hello.txt"

	fstest.WriteToFile(t, []byte("1"), filepath.Join(tempDir, fname1))
	gittest.CommitFilesToGit(t, tempDir)

	paths, err := LsFiles(tempDir, "*.txt")

	require.NoError(t, err)
	require.Empty(t, paths)
}
