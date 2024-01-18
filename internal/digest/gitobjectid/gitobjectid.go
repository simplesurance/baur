package gitobjectid

import (
	"context"

	"github.com/simplesurance/baur/v3/internal/digest"
	"github.com/simplesurance/baur/v3/internal/vcs/git"
)

type PrintfFn func(format string, a ...any)

// Calc calculates git object IDs for files.
// Object IDs are read the first time a digest is requested from the git repository.
type Calc struct {
	repoDir   string
	debugLogf PrintfFn
}

func New(gitRepositoryDir string, debugLogf PrintfFn) *Calc {
	return &Calc{
		repoDir:   gitRepositoryDir,
		debugLogf: debugLogf,
	}
}

// File calculcates the git object ID for the file at absPath.
func (h *Calc) File(absPath string) (*digest.Digest, error) {
	h.debugLogf("gitobjectid: object id not in cache, calculating ID for %q", absPath)
	objectID, err := git.ObjectID(context.TODO(), absPath, h.repoDir)
	if err != nil {
		return nil, err
	}

	return digest.FromStrDigest(objectID, digest.GitObjectID)
}
