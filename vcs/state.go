package vcs

import (
	"path/filepath"
	"sync"

	"github.com/simplesurance/baur/vcs/git"
)

// StateFetcher is an interface for retrieving information about a VCS
// repository.
type StateFetcher interface {
	CommitID() (string, error)
	WorktreeIsDirty() (bool, error)
}

var state = struct {
	path    string
	current StateFetcher
	mu      sync.Mutex
}{}

// GetState returns a git.RepositoryState() if the current directory is in a
// git repository.
// If it's not, *NoVCsState is returned.
// The function caches the result for the last directory it was called.
func GetState(dir string) (StateFetcher, error) {
	state.mu.Lock()
	defer state.mu.Unlock()

	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	if state.current != nil && state.path == dir {
		return state.current, nil
	}

	isGitDir, err := git.IsGitDir(dir)
	if err != nil {
		return nil, err
	}

	if isGitDir {
		state.current = git.NewRepositoryState(dir)

		return state.current, nil
	}

	state.current = &NoVCsState{}

	return state.current, nil
}
