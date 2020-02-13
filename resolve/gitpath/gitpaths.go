package gitpath

import (
	"path/filepath"
	"strings"

	"github.com/simplesurance/baur/fs"
	"github.com/simplesurance/baur/git"
)

// Resolver resolves one or more git glob paths in a git repository by running
// git ls-files.
// Glob path only resolve to files that are tracked in the repository.
type Resolver struct {
	workingDir string
	globs      []string
}

// NewResolver returns a resolver that resolves the passed git glob paths to absolute
// paths
func NewResolver(workingDir string, globs ...string) *Resolver {
	return &Resolver{
		workingDir: workingDir,
		globs:      globs,
	}
}

// Resolve the glob paths to absolute file paths by calling
// git ls-files
func (r *Resolver) Resolve() ([]string, error) {
	out, err := git.LsFiles(r.workingDir, r.globs...)
	if err != nil {
		return nil, err
	}

	relPaths := strings.Split(out, "\n")
	res := make([]string, 0, len(relPaths))

	for _, relPath := range relPaths {
		absPath := filepath.Join(r.workingDir, relPath)

		isFile, err := fs.IsFile(absPath)
		if err != nil {
			return nil, err
		}

		if !isFile {
			continue
		}

		res = append(res, absPath)
	}

	return res, nil
}
