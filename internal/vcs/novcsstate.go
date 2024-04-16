package vcs

import "errors"

var ErrVCSRepositoryNotExist = errors.New("vcs repository not found")

// NoVCsState implements the StateFetcher interface.
// All it's methods return VCSRepositoryNotExistErr.
type NoVCsState struct{}

// CommitID returns VCSRepositoryNotExistErr.
func (*NoVCsState) CommitID() (string, error) {
	return "", ErrVCSRepositoryNotExist
}

// WorktreeIsDirty returns VCSRepositoryNotExistErr.
func (*NoVCsState) WorktreeIsDirty() (bool, error) {
	return false, ErrVCSRepositoryNotExist
}

// WithoutUntracked returns VCSRepositoryNotExistErr.
func (NoVCsState) WithoutUntracked(_ ...string) ([]string, error) {
	return nil, ErrVCSRepositoryNotExist
}

func (*NoVCsState) Name() string {
	return "none"
}

func (*NoVCsState) UntrackedFiles() ([]string, error) {
	return nil, ErrVCSRepositoryNotExist
}
