package fs

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("'%s' does not exist: %w", path, err)
			}

			return fmt.Errorf("%s: %w", path, err)
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

// FindFileInParentDirs finds a file in startPath or its parent directories.
// The function starts looking for a file called filename in startPath and then
// checks recursively its parent directories.
// It returns the absolute path of the first match.
// If it reaches the root directory without finding the file it returns
// os.ErrNotExist.
func FindFileInParentDirs(startPath, filename string) (string, error) {
	// filepath.Clean() is called to remove excessive PathSeperators from the end.
	// If this does not happen, the search might be aborted too early because a path
	// ending in a Separator is interpreted as the root directory.
	searchDir := filepath.Clean(startPath)

	for {
		p := filepath.Join(searchDir, filename)

		_, err := os.Stat(p)
		if err == nil {
			abs, err := filepath.Abs(p)
			if err != nil {
				return "", fmt.Errorf("could not get absolute path of %v: %w", p, err)
			}

			return abs, nil
		}

		if !os.IsNotExist(err) {
			return "", err
		}

		if searchDir[len(searchDir)-1] == os.PathSeparator {
			return "", os.ErrNotExist
		}

		searchDir = filepath.Dir(searchDir)
	}
}

// FindFilesInSubDir returns all directories that contain filename that are in
// searchDir. The function descends up to maxdepth levels of directories below
// searchDir
func FindFilesInSubDir(searchDir, filename string, maxdepth int) ([]string, error) {
	var result []string
	glob := ""

	for i := 0; i <= maxdepth; i++ {
		globPath := filepath.Join(searchDir, glob, filename)

		matches, err := filepath.Glob(globPath)
		if err != nil {
			return nil, err
		}

		for _, m := range matches {
			abs, err := filepath.Abs(m)
			if err != nil {
				return nil, fmt.Errorf("could not get absolute path of %s: %w", m, err)
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
		abs := filepath.Clean(filepath.Join(rootPath, d))
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

// Mkdirs creates directories for all paths and their parent directories if
// they don't exist
func Mkdirs(paths ...string) error {
	for _, p := range paths {
		if err := Mkdir(p); err != nil {
			return err
		}
	}

	return nil
}

// AbsPaths ensures that all elements in paths are absolute paths.
// If an element is not an absolute path, it is joined with rootPath.
func AbsPaths(rootPath string, paths []string) []string {
	result := make([]string, len(paths))

	for i, p := range paths {
		if filepath.IsAbs(p) {
			result[i] = p
			continue
		}

		result[i] = filepath.Join(rootPath, p)
	}

	return result
}

const FileBackupSuffix = ".bak"

// BackupFile renames a file to <OldName><FileBackupSuffix>.
func BackupFile(filepath string) error {
	bakFilePath := filepath + FileBackupSuffix

	return os.Rename(filepath, bakFilePath)
}

// RealPath resolves all symlinks and returns the absolute path.
func RealPath(path string) (string, error) {
	path, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("resolving symlinks in path failed: %w", err)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("computing absolute path of %q failed: %w", path, err)
	}

	return absPath, nil
}
