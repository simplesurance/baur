package gitpath

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v2/internal/exec"
	"github.com/simplesurance/baur/v2/internal/log"
	"github.com/simplesurance/baur/v2/internal/testutils/fstest"
	"github.com/simplesurance/baur/v2/internal/testutils/gittest"
)

func TestGitPathResolverIgnoresUntrackedFiles(t *testing.T) {
	log.StdLogger.SetOutput(log.NewTestLogOutput(t))
	exec.DefaultDebugfFn = t.Logf

	gitDir := t.TempDir()
	gittest.CreateRepository(t, gitDir)

	fstest.WriteToFile(t, []byte("123"), filepath.Join(gitDir, "subdir", "file1.txt"))
	gittest.CommitFilesToGit(t, gitDir)

	fstest.WriteToFile(t, []byte("123"), filepath.Join(gitDir, "subdir", "file2.txt"))

	gitResolver := &Resolver{}
	gitResult, err := gitResolver.Resolve(gitDir, filepath.Join(gitDir, "subdir", "*"))
	require.NoError(t, err)
	require.NotEmpty(t, gitResult)

	assert.ElementsMatch(t, []string{filepath.Join(gitDir, "subdir", "file1.txt")}, gitResult)
}
