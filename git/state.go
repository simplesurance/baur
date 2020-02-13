package git

import (
	"sync"
)

// RepositoryState lazyLoads and caches the commitID and worktree state of a Git repository.
type RepositoryState struct {
	path string

	lock            sync.Mutex
	commitID        *string
	worktreeIsDirty *bool
}

// NewRepositoryState initializes a RepositoryState for the given git repository.
func NewRepositoryState(repositoryPath string) *RepositoryState {
	return &RepositoryState{
		path: repositoryPath,
	}
}

// GitCommitID calls git.CommitID() for the repository.
// After the first successful call the commit ID is stored and the stored value
// is returned on successive calls.
func (g *RepositoryState) CommitID() (string, error) {
	g.lock.Lock()
	defer g.lock.Unlock()

	if g.commitID == nil {
		commitID, err := CommitID(g.path)
		if err != nil {
			return "", err
		}

		g.commitID = &commitID
	}

	return *g.commitID, nil
}

// WorktreeIsDirty calls git.WorktreeIsDirty.
// After the first successful call the result is stored and the stored value is
// returned on successive calls.
func (g *RepositoryState) WorktreeIsDirty() (bool, error) {
	g.lock.Lock()
	defer g.lock.Unlock()

	if g.worktreeIsDirty == nil {
		isDirty, err := WorktreeIsDirty(g.path)
		if err != nil {
			return false, err
		}

		g.worktreeIsDirty = &isDirty
	}

	return *g.worktreeIsDirty, nil
}
