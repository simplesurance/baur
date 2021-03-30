// Package fstest provides test utilties to operate with files and directories
package fstest

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

// WriteToFile writes data to a file.
// Directories that are in the path but do not exist are created.
// If an error happens, t.Fatal() is called.
func WriteToFile(t *testing.T, data []byte, path string) {
	t.Helper()

	dir := filepath.Dir(path)

	err := os.MkdirAll(dir, 0775)
	if err != nil {
		t.Fatal(err)
	}

	err = ioutil.WriteFile(path, data, 0644)
	if err != nil {
		t.Fatal(err)
	}
}

// Chmod is wrappoer of os.Chmod that fails the test if chmod returns an error.
func Chmod(t *testing.T, name string, mode os.FileMode) {
	t.Helper()

	if err := os.Chmod(name, mode); err != nil {
		t.Fatal(err)
	}
}

// TempDir returns a path, with all symlinks in it resolved, to a unique
// temporary directory.
// The directory is removed via t.Cleanup() on termination of the test.
// (On MacOS t.TempDir() returns a symlink.)
func TempDir(t *testing.T) string {
	t.Helper()

	p, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal("failed to resolve symlinks in tempdir path:", err)
	}

	return p
}
