package main

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v2/internal/exec"
	"github.com/simplesurance/baur/v2/internal/log"
	"github.com/simplesurance/baur/v2/internal/resolve/gitpath"
	"github.com/simplesurance/baur/v2/internal/resolve/glob"
	"github.com/simplesurance/baur/v2/internal/testutils/fstest"
	"github.com/simplesurance/baur/v2/internal/testutils/gittest"
)

func TestFilesAndGitFilesPatternBehaveTheSame(t *testing.T) {
	log.StdLogger.SetOutput(log.NewTestLogOutput(t))
	exec.DefaultDebugfFn = t.Logf

	gitDir := fstest.TempDir(t)

	fstest.WriteToFile(t, []byte("123"), filepath.Join(gitDir, "subdir", "file1.txt"))
	fstest.WriteToFile(t, []byte("123"), filepath.Join(gitDir, "subdir", "file2.txt"))
	fstest.WriteToFile(t, []byte("123"), filepath.Join(gitDir, "sub", "subdir", "file.txt"))

	gittest.CreateRepository(t, gitDir)
	gittest.CommitFilesToGit(t, gitDir)

	gitResolver := &gitpath.Resolver{}
	globResolver := &glob.Resolver{}

	testPatterns := []string{
		filepath.Join("sub", "**"),
		"subdir*",
		filepath.Join("subdir", "*"),
		filepath.Join("subdir", "file*"),
		filepath.Join("subdir", "file?.txt"),
		filepath.Join("subdir", "*.txt"),
	}

	for _, pattern := range testPatterns {
		t.Run(pattern, func(t *testing.T) {
			gitResult, err := gitResolver.Resolve(gitDir, filepath.Join(gitDir, pattern))
			require.NoError(t, err, "gitresolver failed")

			globResult, err := globResolver.Resolve(filepath.Join(gitDir, pattern))
			require.NoError(t, err, "globresolver failed")

			assert.ElementsMatch(t, gitResult, globResult, "gitpath and glob resolver did not resolve to same files for same pattern")
		})
	}
}
