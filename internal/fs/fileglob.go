package fs

import (
	"errors"
	"os"

	"github.com/bmatcuk/doublestar/v4"
)

// FileGlob resolves the pattern to file paths.
// If the pattern is an absolute path, absolute paths are returned, otherwise
// relative paths.
// Files are resolved in the same way then filepath.Glob() does, with the
// following exceptions:
//   - it also supports '**' to match files and directories recursively,
//   - it only returns paths to files, no directory paths,
//   - if a part of the pattern is a path and it does not exist an error that
//     can be tested with os.IsNotExist() is returned.
//
// If a globPath doesn't match any files an empty []string is returned and
// error is nil
func FileGlob(pattern string) ([]string, error) {
	globRes, err := doublestar.FilepathGlob(
		pattern,
		doublestar.WithFailOnIOErrors(),
		doublestar.WithFailOnPatternNotExist(),
	)
	if err != nil {
		if errors.Is(err, doublestar.ErrPatternNotExist) {
			return nil, os.ErrNotExist
		}
		return nil, err
	}

	res := make([]string, 0, len(globRes))
	for _, path := range globRes {
		isFile, err := IsFile(path)
		if err != nil {
			return nil, err
		}

		if !isFile {
			continue
		}

		res = append(res, path)
	}

	return res, err
}

func MatchGlob(pattern, path string) (bool, error) {
	return doublestar.PathMatch(pattern, path)
}
