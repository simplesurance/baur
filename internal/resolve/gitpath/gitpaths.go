package gitpath

import (
	"fmt"
	"path/filepath"

	"github.com/simplesurance/baur/v2/internal/fs"
	"github.com/simplesurance/baur/v2/internal/vcs/git"
)

// Resolver resolves one or more git glob paths in a git repository by running
// git ls-files.
// Glob paths are only resolved to files that exist in the filesystem and are tracked in the repository.
type Resolver struct{}

// Resolve resolves the glob paths to absolute file paths by calling git ls-files.
// workingDir must be a directory that is part of a Git repository.
// Glob must be a pattern that is an absolute path.
// If a glob does not resolve to an existing file in the filesystem or the file
// is not part of the git repository an empty slice is returned.
func (r *Resolver) Resolve(workingDir, glob string) ([]string, error) {
	if !filepath.IsAbs(glob) {
		return nil, fmt.Errorf("%s is not an absolute glob path", glob)
	}
	// Globs are resolved via the same method then resolve/files
	// uses. This ensures the same pattern resolves in the same way
	// for both. git ls-files resolves '*` differently, it also matches directory seperators.
	paths, err := fs.FileGlob(glob)
	if err != nil {
		return nil, fmt.Errorf("resolving %q failed: %w", glob, err)
	}

	if len(paths) == 0 {
		return []string{}, nil
	}

	relPaths, err := git.LsFiles(workingDir, paths...)
	if err != nil {
		return nil, fmt.Errorf("git ls-files failed: %w", err)
	}

	res := make([]string, 0, len(relPaths))
	for _, relPath := range relPaths {
		absPath := filepath.Join(workingDir, relPath)
		res = append(res, absPath)
	}

	return res, nil
}
