package glob

import (
	"fmt"

	"github.com/simplesurance/baur/v2/internal/fs"
)

// Resolver resolves a glob path to files. The functionality is the same then
// filepath.Glob() with the addition that '**' is supported to match files
// directories recursively.
type Resolver struct{}

// Resolve resolves the globPath to absolute file paths.
// Files are resolved in the same way then filepath.Glob() does, with 2 Exceptions:
// - it also supports '**' to match files and directories recursively,
// - it only returns paths to files, no directory paths,
// If a globPath doesn't match any files an empty []string is returned and
// error is nil
func (r *Resolver) Resolve(globPath string) ([]string, error) {
	paths, err := fs.FileGlob(globPath)
	if err != nil {
		return nil, fmt.Errorf("resolving %q failed: %w", globPath, err)
	}

	return paths, nil
}
