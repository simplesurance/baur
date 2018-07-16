package fs

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

// IsFile returns true if path is a file.
// If the path does not exist an error is returned
func IsFile(path string) (bool, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return false, err
	}

	return !fi.IsDir(), nil
}

// FileExists returns true if path exist and is a file
func FileExists(path string) bool {
	ret, _ := IsFile(path)

	return ret
}

// DirsExist runs DirExists for multiple paths.
func DirsExist(paths []string) error {
	for _, path := range paths {
		isDir, err := IsDir(path)
		if os.IsNotExist(err) {
			return fmt.Errorf("'%s' does not exist", path)
		}

		if !isDir {
			return fmt.Errorf("'%s' is not a directory", path)
		}
	}

	return nil
}

// IsDir returns true if the path is a directory.
func IsDir(path string) (bool, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return false, err
	}

	return fi.IsDir(), nil
}

// FindFileInParentDirs finds a directory that contains filename. The function
// starts searching in startPath and then checks recursively each parent
// directory for the file. It returns the absolute path to the first found
// directory contains the file.
// If it reaches the root directory without finding the file it returns
// os.ErrNotExist
func FindFileInParentDirs(startPath, filename string) (string, error) {
	searchDir := startPath

	for {
		p := path.Join(searchDir, filename)

		_, err := os.Stat(p)
		if err == nil {
			abs, err := filepath.Abs(p)
			if err != nil {
				return "", errors.Wrapf(err,
					"could not get absolute path of %v", p)
			}

			return abs, nil
		}

		if !os.IsNotExist(err) {
			return "", err
		}

		// TODO: how to detect OS independent if reached the root dir
		if searchDir == "/" {
			return "", os.ErrNotExist
		}

		searchDir = path.Join(searchDir, "..")
	}
}

// FindFilesInSubDir returns all directories that contain filename that are in
// searchDir. The function descends up to maxdepth levels of directories below
// searchDir
func FindFilesInSubDir(searchDir, filename string, maxdepth int) ([]string, error) {
	var result []string
	glob := ""

	for i := 0; i <= maxdepth; i++ {
		globPath := path.Join(searchDir, glob, filename)

		matches, err := filepath.Glob(globPath)
		if err != nil {
			return nil, err
		}

		for _, m := range matches {
			abs, err := filepath.Abs(m)
			if err != nil {
				return nil, errors.Wrapf(err, "could not get absolute path of %s", m)
			}

			result = append(result, abs)
		}

		glob += "*/"
	}

	return result, nil
}

// FindAllDirs returns recursively all diretories in path, including the
// passed path dir
func FindAllDirs(path string) ([]string, error) {
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

func findAllDirsNoDups(result map[string]struct{}, path string) error {
	isDir, err := IsDir(path)
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

// PathsJoin returns a list where all paths in relPaths are prefixed with
// rootPath
func PathsJoin(rootPath string, relPaths []string) []string {
	absPaths := make([]string, 0, len(relPaths))

	for _, d := range relPaths {
		abs := path.Clean(path.Join(rootPath, d))
		absPaths = append(absPaths, abs)
	}

	return absPaths
}

// FileReadLine reads the first line from a file
func FileReadLine(path string) (string, error) {
	fd, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer fd.Close()

	r := bufio.NewReader(fd)
	content, err := r.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}

	return content, nil
}

// FileSize returns the size of a file in Bytes
func FileSize(path string) (int64, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return -1, err
	}

	return stat.Size(), nil
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

	dirs, err := FindAllDirs(basePath)
	if err != nil {
		return nil, err
	}

	for i := range dirs {
		dirs[i] = filepath.Join(dirs[i], glob)
	}

	return dirs, nil
}

// Glob is similar then filepath.Glob() but also support '**' to match files
// and directories  recursively and only returns paths to Files.
// If a Glob doesn't match any files an empty []string is returned and error is
// nil
func Glob(path string) ([]string, error) {
	var globPaths []string

	if strings.Contains(path, "**") {
		expandedPaths, err := expandDoubleStarGlob(path)
		if err != nil {
			return nil, errors.Wrap(err, "expanding '**' failed")
		}

		globPaths = expandedPaths
	} else {
		globPaths = []string{path}
	}

	// capacity could be bigger to have some memory  available for the
	// elements returned by filepath.Glob() that will be added
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
		isFile, err := IsFile(p)
		if err != nil {
			return nil, errors.Wrapf(err, "resolved path %q does not exist", p)
		}

		if isFile {
			res = append(res, p)
		}
	}

	return res, nil
}
