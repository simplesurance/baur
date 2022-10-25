package vcs

import (
	"path/filepath"
	"sync"

	"github.com/simplesurance/baur/v3/internal/vcs/git"
)

// StateFetcher is an interface for retrieving information about a VCS
// repository.
type StateFetcher interface {
	CommitID() (string, error)
	WorktreeIsDirty() (bool, error)
	WithoutUntracked(paths ...string) ([]string, error)
}

var state = struct {
	path    string
	current StateFetcher
	mu      sync.Mutex
}{}

type Logfn func(format string, v ...interface{})

// GetState returns a git.RepositoryState() if the current directory is in a
// git repository and git command is in $PATH.
// If it's not, *NoVCsState is returned.
// The function caches the result for the last directory it was called.
// logfunc can be nil, to disable logging
func GetState(dir string, logfunc Logfn) (StateFetcher, error) {
	state.mu.Lock()
	defer state.mu.Unlock()

	if logfunc == nil {
		logfunc = func(_ string, _ ...interface{}) {}
	}

	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	if state.current != nil && state.path == dir {
		return state.current, nil
	}

	if !git.CommandIsInstalled() {
		logfunc("vcs: git support disabled, git command is not installed or not in $PATH\n")

		state.current = &NoVCsState{}
		return state.current, nil
	}

	isGitDir, err := git.IsGitDir(dir)
	if err != nil {
		return nil, err
	}

	if isGitDir {
		logfunc("vcs: %s is part of a git repository found\n", dir)

		state.current = git.NewRepository(dir)

		return state.current, nil
	}

	logfunc("vcs: git support disabled, %s is not part of a git repository\n", dir)

	state.current = &NoVCsState{}

	return state.current, nil
}
