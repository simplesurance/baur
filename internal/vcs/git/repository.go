package git

import (
	"errors"
	"path/filepath"
	"sync"

	"github.com/simplesurance/baur/v4/internal/set"
)

const Name = "git"

// ErrRepositoryNotFound is returned when a directory is not part of a git
// repository.
var ErrRepositoryNotFound = errors.New("git repository not found")

// Repository reads information from a Git repository.
type Repository struct {
	path string

	lock            sync.Mutex
	commitID        *string
	worktreeIsDirty *bool
	untrackedFiles  set.Set[string]
}

// NewRepositoryWithCheck returns a new Repository object for the git repository at dir.
// If dir is not part of a git repository, the "git" command is not installed
// or can not be located in $PATH an error is returned.
func NewRepositoryWithCheck(dir string) (*Repository, error) {
	if !CommandIsInstalled() {
		return nil, errors.New("git command not found, ensure git is installed and in $PATH")
	}

	isGitDir, err := IsGitDir(dir)
	if err != nil {
		return nil, err
	}

	if !isGitDir {
		return nil, ErrRepositoryNotFound
	}

	return NewRepository(dir), nil
}

// NewRepository returns a new Repository object that interacts with the git
// repository in dir.
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

		if g.untrackedFiles.Contains(relPath) {
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
		g.untrackedFiles = set.From(files)
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
