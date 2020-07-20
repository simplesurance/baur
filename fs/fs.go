package fs

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/errors"
)

// IsFile returns true if path is a file.
// If the path does not exist an error is returned
func IsFile(path string) (bool, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return false, err
	}

	return fi.Mode().IsRegular(), nil
}

// FileExists returns true if path exist and is a file
func FileExists(path string) bool {
	ret, _ := IsFile(path)

	return ret
}

// DirsExist runs DirExists for multiple paths.
func DirsExist(paths ...string) error {
	for _, path := range paths {
		isDir, err := IsDir(path)
		if os.IsNotExist(err) {
			return fmt.Errorf("'%s' does not exist: %w", path, err)
		}

		if !isDir {
			return fmt.Errorf("'%s' is not a directory", path)
		}
	}

	return nil
}

// IsDir returns true if the path is a directory.
// If the directory does not exist, the error from os.Stat() is returned.
func IsDir(path string) (bool, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return false, err
	}

	return fi.IsDir(), nil
}

// IsRegularFile returns true if path is a regular file.
// If the directory does not exist, the error from os.Stat() is returned.
func IsRegularFile(path string) (bool, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return false, err
	}

	return fi.Mode().IsRegular(), nil
}

// SameFile calls os.Samefile(), if one of the files does not exist, the error
// from os.Stat() is returned.
func SameFile(a, b string) (bool, error) {
	aFi, err := os.Stat(a)
	if err != nil {
		return false, err
	}

	bFi, err := os.Stat(b)
	if err != nil {
		return false, err
	}

	return os.SameFile(aFi, bFi), nil
}

// FindFileInParentDirs finds a file in startPath or it's parent directories.
// The function starts looking for a file called filename in startPath and then
// checks recursively it's parent directors.
// It returns the absolute path of the first match.
// If it reaches the root directory without finding the file it returns
// os.ErrNotExist.
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

// Mkdir creates recursively directories
func Mkdir(path string) error {
	return os.MkdirAll(path, os.FileMode(0755))
}

// AbsPaths prepends to all paths in relPaths the passed rootPath.
func AbsPaths(rootPath string, relPaths []string) []string {
	result := make([]string, 0, len(rootPath))

	for _, relPath := range relPaths {
		absPath := filepath.Join(rootPath, relPath)
		result = append(result, absPath)
	}

	return result
}
