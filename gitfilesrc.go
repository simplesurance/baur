package baur

import (
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/simplesurance/baur/fs"
	"github.com/simplesurance/baur/git"
)

//GitPaths resolves multiple git filepath patterns to paths in the filesystem.
type GitPaths struct {
	baseDir string
	paths   []string
}

// NewGitPaths returns a new GitPaths
func NewGitPaths(baseDir string, paths []string) *GitPaths {
	return &GitPaths{
		baseDir: baseDir,
		paths:   paths,
	}
}

// Resolve returns a list of files that are matching it's path
func (g *GitPaths) Resolve() ([]BuildInput, error) {
	arg := strings.Join(g.paths, " ")
	out, err := git.LsFiles(g.baseDir, arg)
	if err != nil {
		return nil, err
	}

	paths := strings.Split(out, "\n")
	res := make([]BuildInput, 0, len(paths))

	for _, p := range paths {
		isFile, err := fs.IsFile(filepath.Join(g.baseDir, p))
		if err != nil {
			return nil, errors.Wrapf(err, "resolved path %q does not exist", p)
		}

		if isFile {
			res = append(res, NewFile(g.baseDir, p))
		}
	}

	return res, nil
}
