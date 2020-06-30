package gitpath

import (
	"path/filepath"
	"strings"

	"github.com/simplesurance/baur/fs"
	"github.com/simplesurance/baur/vcs/git"
)

// Resolver resolves one or more git glob paths in a git repository by running
// git ls-files.
// Glob paths are only resolved to files that are tracked in the repository.
type Resolver struct{}

// Resolve resolves the glob paths to absolute file paths by calling git ls-files.
// workingDir must be a directory that is part of a Git repository.
// If a resolved file does not exist an error is returned.
func (r *Resolver) Resolve(workingDir string, globs ...string) ([]string, error) {
	if len(globs) == 0 {
		return []string{}, nil
	}

	out, err := git.LsFiles(workingDir, globs...)
	if err != nil {
		return nil, err
	}

	relPaths := strings.Split(out, "\n")
	res := make([]string, 0, len(relPaths))

	for _, relPath := range relPaths {
		absPath := filepath.Join(workingDir, relPath)

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
