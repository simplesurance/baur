package gitpath

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/simplesurance/baur/v1/internal/fs"
	"github.com/simplesurance/baur/v1/internal/vcs/git"
)

// Resolver resolves one or more git glob paths in a git repository by running
// git ls-files.
// Glob paths are only resolved to files that are tracked in the repository.
type Resolver struct{}

// Resolve resolves the glob paths to absolute file paths by calling git ls-files.
// workingDir must be a directory that is part of a Git repository.
// If a resolved file does not exist an error is returned.
func (r *Resolver) Resolve(workingDir string, errorUnmatch bool, globs ...string) ([]string, error) {
	var resolvedPaths []string

	for _, glob := range globs {
		absGlob := filepath.Join(workingDir, glob)
		// Globs are resolved via the same method then resolve/files
		// uses. This ensures the same pattern resolves in the same way
		// for both. git ls-files resolves '*` differently, it also matches directory seperators.
		paths, err := fs.FileGlob(absGlob)
		if err != nil {
			return nil, fmt.Errorf("resolving %q failed: %w", absGlob, err)
		}

		if len(paths) == 0 {
			if errorUnmatch {
				return nil, fmt.Errorf("%q did not match any files: %w", absGlob, os.ErrNotExist)
			}

			continue
		}

		resolvedPaths = append(resolvedPaths, paths...)
	}

	if len(resolvedPaths) == 0 {
		return resolvedPaths, nil
	}

	relPaths, err := git.LsFiles(workingDir, resolvedPaths...)
	if err != nil {
		return nil, fmt.Errorf("git ls-files failed: %w", err)
	}

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
