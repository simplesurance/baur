package git

import (
	"path/filepath"
	"sync"
)

const Name = "git"

// Repository reads information from a Git repository.
type Repository struct {
	path string

	lock            sync.Mutex
	commitID        *string
	worktreeIsDirty *bool
	untrackedFiles  map[string]struct{}
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

func (g *Repository) WithoutUntracked(paths ...string) ([]string, error) {
	// res will be often smaller then paths, we allocate the maximum
	// possible size and trade more memory usage against less malloc calls.
	res := make([]string, 0, len(paths))

	if err := g.initUntrackedFiles(); err != nil {
		return nil, err
	}

	for _, p := range paths {
		var relPath string
		if filepath.IsAbs(p) {
			var err error
			relPath, err = filepath.Rel(g.path, p)
			if err != nil {
				return nil, err
			}
		} else {
			relPath = p
		}

		if _, isUntracked := g.untrackedFiles[relPath]; isUntracked {
			continue
		}

		res = append(res, p)
	}

	return res, nil
}

func (g *Repository) initUntrackedFiles() error {
	g.lock.Lock()
	defer g.lock.Unlock()

	if g.untrackedFiles == nil {
		files, err := UntrackedFiles(g.path)
		if err != nil {
			return err
		}

		m := make(map[string]struct{}, len(files))

		for _, p := range files {
			m[p] = struct{}{}
		}
		g.untrackedFiles = m
	}

	return nil
}

// UntrackedFiles returns a list of untracked and modified files in the git repository.
// Files that exist and are in a .gitignore file are included.
func (g *Repository) UntrackedFiles() ([]string, error) {
	return UntrackedFiles(g.path)
}

func (g *Repository) Name() string {
	return Name
}
