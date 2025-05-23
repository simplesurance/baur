// Package fstest provides test utilties to operate with files and directories
package fstest

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// WriteToFile writes data to the file at path.
// The file is created with 0755 permissions.
// Directories that are in the path but do not exist are created.
// If an error happens, t.Fatal() is called.
func WriteExecutable(t *testing.T, data []byte, path string) {
	t.Helper()
	writeToFile(t, data, path, 0o755)
}

// WriteToFile writes data to a file.
// Directories that are in the path but do not exist are created.
// If an error happens, t.Fatal() is called.
func WriteToFile(t *testing.T, data []byte, path string) {
	t.Helper()
	writeToFile(t, data, path, 0o644)
}

func writeToFile(t *testing.T, data []byte, path string, perm fs.FileMode) {
	dir := filepath.Dir(path)

	MkdirAll(t, dir)

	err := os.WriteFile(path, data, perm)
	if err != nil {
		t.Fatal(err)
	}
}

func MkdirAll(t *testing.T, path string) {
	err := os.MkdirAll(path, 0o775)
	if err != nil {
		t.Fatal(err)
	}
}

func Symlink(t *testing.T, oldname, newname string) {
	t.Helper()

	err := os.Symlink(oldname, newname)
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

// ReadFile reads the file at path and returns its content.
func ReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading file %q failed: %s", path, err)
	}

	return data
}
