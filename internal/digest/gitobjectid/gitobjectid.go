package gitobjectid

import (
	"context"

	"github.com/simplesurance/baur/v4/internal/digest"
	"github.com/simplesurance/baur/v4/internal/vcs/git"
)

type PrintfFn func(format string, a ...any)

// Hash calculates git object IDs for files.
// Object IDs are read the first time a digest is requested from the git repository.
type Hash struct {
	repoDir   string
	debugLogf PrintfFn
}

func New(gitRepositoryDir string, debugLogf PrintfFn) *Hash {
	return &Hash{
		repoDir:   gitRepositoryDir,
		debugLogf: debugLogf,
	}
}

// File calculcates the git object ID for the file at absPath.
func (h *Hash) File(absPath string) (*digest.Digest, error) {
	h.debugLogf("gitobjectid: object id not in cache, calculating ID for %q", absPath)
	objectID, err := git.ObjectID(context.TODO(), absPath, h.repoDir)
	if err != nil {
		return nil, err
	}

	return digest.FromStrDigest(objectID, digest.GitObjectID)
}
