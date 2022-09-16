package fs

import (
	"fmt"
	"path/filepath"

	"github.com/bmatcuk/doublestar/v4"
)

// FileGlob resolves the pattern to absolute file paths.
// Files are resolved in the same way then filepath.Glob() does, with  Exceptions:
// - it also supports '**' to match files and directories recursively,
// - it only returns paths to files, no directory paths,
// If a globPath doesn't match any files an empty []string is returned and
// error is nil
func FileGlob(pattern string) ([]string, error) {
	globRes, err := doublestar.FilepathGlob(pattern, doublestar.WithFailOnIOErrors())
	if err != nil {
		return nil, err
	}

	res := make([]string, 0, len(globRes))
	for _, r := range globRes {
		absPath, err := filepath.Abs(r)
		if err != nil {
			return nil, fmt.Errorf("evaluating absolute path of %q failed: %w", absPath, err)
		}

		isFile, err := IsFile(absPath)
		if err != nil {
			return nil, err
		}

		if !isFile {
			continue
		}
		res = append(res, absPath)
	}

	return res, err
}
