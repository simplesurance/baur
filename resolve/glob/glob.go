package glob

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/simplesurance/baur/fs"
)

// Resolver resolves a glob path to files. The functionality is the same then
// filepath.Glob() with the addition that '**' is supported to match files
// directories recursively.
type Resolver struct {
	glob string
}

// NewResolver returns a resolver that resolves glob relative to path
func NewResolver(glob string) *Resolver {
	return &Resolver{
		glob: glob,
	}
}

// Resolve returns absolute paths to files that specify the glob path
// glob does the same  then filepath.Glob() with 2 Exceptions:
//- it also supports '**' to match files and directories recursively
//- it and only returns paths to files, no directory paths
// If a Glob doesn't match any files an empty []string is returned and error is
// nil
func (r *Resolver) Resolve() ([]string, error) {
	var globPaths []string

	if strings.Contains(r.glob, "**") {
		expandedPaths, err := expandDoubleStarGlob(r.glob)
		if err != nil {
			return nil, errors.Wrap(err, "expanding '**' failed")
		}

		globPaths = expandedPaths
	} else {
		globPaths = []string{r.glob}
	}

	paths := make([]string, 0, len(globPaths))
	for _, globPath := range globPaths {
		path, err := filepath.Glob(globPath)
		if err != nil {
			return nil, err
		}

		if path == nil {
			continue
		}

		paths = append(paths, path...)
	}

	res := make([]string, 0, len(paths))
	for _, p := range paths {
		isFile, err := fs.IsFile(p)
		if err != nil {
			return nil, errors.Wrapf(err, "resolved path %q does not exist", p)
		}

		if isFile {
			res = append(res, p)
		}
	}

	return res, nil
}

func findAllDirsNoDups(result map[string]struct{}, path string) error {
	isDir, err := fs.IsDir(path)
	if err != nil {
		return errors.Wrapf(err, "IsDir(%s) failed", path)
	}

	if !isDir {
		return nil
	}

	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		return errors.Wrapf(err, "resolving symlinks in %q failed", path)
	}

	if _, exist := result[path]; exist {
		return nil
	}
	result[path] = struct{}{}

	globPath := filepath.Join(path, "*")
	rootGlob, err := filepath.Glob(globPath) // is filepath.Walk() faster?
	if err != nil {
		return errors.Wrapf(err, "glob of %q failed", globPath)
	}

	for _, path := range rootGlob {
		err = findAllDirsNoDups(result, path)
		if err != nil {
			return err
		}
	}

	return nil
}

// findAllDirs returns recursively all diretories in path, including the
// passed path dir
func findAllDirs(path string) ([]string, error) {
	resultMap := map[string]struct{}{}

	err := findAllDirsNoDups(resultMap, path)
	if err != nil {
		return nil, err
	}

	res := make([]string, 0, len(resultMap))
	for p := range resultMap {
		res = append(res, p)
	}

	return res, nil
}

// expandDoubleStarGlob takes a glob path containing  '**' and returns a list of
// paths were ** is expanded recursively to all matching directories.  If '**'
// is the last part in the path, the returned paths will end in '/*' to glob
// match all files in those directories
func expandDoubleStarGlob(absGlobPath string) ([]string, error) {
	spl := strings.Split(absGlobPath, "**")
	if len(spl) < 2 {
		return nil, fmt.Errorf("%q does not contain '**'", absGlobPath)
	}

	basePath := spl[0]
	glob := spl[1]

	if len(glob) == 0 {
		glob = "*"
	}

	dirs, err := findAllDirs(basePath)
	if err != nil {
		return nil, err
	}

	for i := range dirs {
		dirs[i] = filepath.Join(dirs[i], glob)
	}

	return dirs, nil
}
