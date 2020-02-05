// Package fstest provides test utilties to operate with files and directories
package fstest

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

// CreateTempDir creates a new temporary directory, returns a name and a cleanup
// function that removes the directory.
func CreateTempDir(t *testing.T) (string, func()) {
	t.Helper()

	dir, err := ioutil.TempDir("", "baur-filesrc-test")
	if err != nil {
		t.Fatal(err)
	}

	return dir, func() { os.RemoveAll(dir) }
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
