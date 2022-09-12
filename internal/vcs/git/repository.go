package git

import (
	"sync"
)

// Repository reads information from a Git repository.
type Repository struct {
	path string

	lock            sync.Mutex
	commitID        *string
	worktreeIsDirty *bool
}

// NewRepository creates a new Repository that interacts with the repository at
// dir. dir must be directory in a Git worktree. This means it must have a
// .git/ directory or one of the parent directories must contain a .git/
// directory.
func NewRepository(dir string) *Repository {
	return &Repository{
		path: dir,
	}
}

// CommitID calls git.CommitID() for the repository.
// After the first successful call the commit ID is stored and the stored value
// is returned on successive calls.
func (g *Repository) CommitID() (string, error) {
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
func (g *Repository) WorktreeIsDirty() (bool, error) {
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
