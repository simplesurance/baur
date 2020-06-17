// Package fstest provides test utilties to operate with files and directories
package fstest

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

// CreateTempDir creates a new temporary directory.
// Removal of the directory is registered with t.Cleanup().
// The function returns the path of the temporary directory.
func CreateTempDir(t *testing.T) string {
	t.Helper()

	dir, err := ioutil.TempDir("", "baur-filesrc-test")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { os.RemoveAll(dir) })

	return dir
}

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
