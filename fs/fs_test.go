package fs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/simplesurance/baur/testutils/fstest"
	"github.com/simplesurance/baur/testutils/strtest"
)

func Test_FindAllSubDirs(t *testing.T) {
	tempdir, cleanupFunc := fstest.CreateTempDir(t)
	defer cleanupFunc()

	expectedResults := []string{
		tempdir,
		filepath.Join(tempdir, "1"),
		filepath.Join(tempdir, "1/2"),
		filepath.Join(tempdir, "1/2/3/"),
	}

	err := os.MkdirAll(filepath.Join(tempdir, "1/2/3"), os.ModePerm)
	if err != nil {
		t.Fatal("creating subdirectories failed:", err)
	}

	res, err := FindAllDirs(tempdir)
	if err != nil {
		t.Fatal(err)
	}

	if len(res) != len(expectedResults) {
		t.Errorf("unexpected number of elements returned, expected: %q, got: %q",
			expectedResults, res)
	}

	for _, er := range expectedResults {
		if !strtest.InSlice(res, er) {
			t.Errorf("%q is missing in result %q", er, res)
		}
	}

	return
}
