package baur

import (
	"path/filepath"
	"strings"

	"github.com/simplesurance/baur/fs"
	"github.com/simplesurance/baur/git"
)

//GitPaths resolves multiple git filepath patterns to paths in the filesystem.
type GitPaths struct {
	repositoryRootPath string
	relAppPath         string
	paths              []string
}

// NewGitPaths returns a new GitPaths
func NewGitPaths(repositoryRootPath, relAppPath string, gitPaths []string) *GitPaths {
	return &GitPaths{
		repositoryRootPath: repositoryRootPath,
		relAppPath:         relAppPath,
		paths:              gitPaths,
	}
}

// Resolve returns a list of files that are matching it's path
func (g *GitPaths) Resolve() ([]BuildInput, error) {
	baseDir := filepath.Join(g.repositoryRootPath, g.relAppPath)

	arg := strings.Join(g.paths, " ")
	out, err := git.LsFiles(baseDir, arg)
	if err != nil {
		return nil, err
	}

	paths := strings.Split(out, "\n")
	res := make([]BuildInput, 0, len(paths))

	for _, p := range paths {
		isFile, err := fs.IsFile(filepath.Join(baseDir, p))
		if err != nil {
			return nil, err
		}

		if isFile {
			res = append(res, NewFile(g.repositoryRootPath, filepath.Join(g.relAppPath, p)))
		}
	}

	return res, nil
}

// Type returns the type of resolver
func (g *GitPaths) Type() string {
	return "GitFiles"
}

// String returns the path to resolve
func (g *GitPaths) String() string {
	return strings.Join(g.paths, ", ")
}
