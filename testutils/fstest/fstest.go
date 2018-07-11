// Package fstest provides test utilties to operate with files and directories
package fstest

import (
	"io/ioutil"
	"os"
	"testing"
)

// CreateTempDir creates a new temporary directory, returns a name and a cleanup
// function that removes the directory.
func CreateTempDir(t *testing.T) (string, func()) {
	dir, err := ioutil.TempDir("", "baur-filesrc-test")
	if err != nil {
		t.Fatal(err)
	}

	return dir, func() { os.RemoveAll(dir) }
}
