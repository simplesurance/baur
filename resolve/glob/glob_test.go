package glob

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

	res, err := findAllDirs(tempdir)
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
}

func createFiles(t *testing.T, basedir string, paths []string) {
	for _, p := range paths {
		fullpath := filepath.Join(basedir, p)
		f, err := os.Create(fullpath)
		if err != nil {
			t.Fatal("creating file failed:", err)
		}

		f.Close()
	}
}

func checkFilesInResolvedFiles(t *testing.T, tempdir string, resolvedFiles []string, tc *testcase) {
	if len(resolvedFiles) != len(tc.expectedMatches) {
		t.Errorf("resolved to %d files (%v), expected %d (%+v)",
			len(resolvedFiles), resolvedFiles,
			len(tc.expectedMatches), tc)
	}

	for _, e := range resolvedFiles {
		relPath, err := filepath.Rel(tempdir, e)
		if err != nil {
			t.Errorf("getting Relpath of %q to %q failed", e, tempdir)
		}

		if !strtest.InSlice(tc.expectedMatches, relPath) {
			t.Errorf("%q (%q) was returned but is not in expected return slice (%+v), testcase: %+v",
				e, relPath, tc.expectedMatches, tc)
		}
	}
}

type testcase struct {
	files           []string
	dir             string
	expectedMatches []string
	fileSrcGlobPath string
}

func Test_Resolve(t *testing.T) {
	testcases := []*testcase{
		{
			files: []string{
				"a.go",
				"hello.go",
				"bla-blub.go.go",
				"thisnot",
				"notme.og",
			},
			expectedMatches: []string{
				"a.go",
				"hello.go",
				"bla-blub.go.go",
			},
			fileSrcGlobPath: "*.go",
		},

		{
			files: []string{
				"thisnot",
				"1/notme.og",
				"hello.bat",
				"yo.bat",
			},
			dir: "1",
			expectedMatches: []string{
				"hello.bat",
				"yo.bat",
			},
			fileSrcGlobPath: "*.b??",
		},

		{
			files: []string{
				"1/2/3/a.go",
				"1/hello.go",
				"bla-blub.go.go",
				"1/2/yo.go.go",
				"thisnot",
				"1/notme.og",
			},
			dir: "1/2/3",
			expectedMatches: []string{
				"1/2/3/a.go",
				"1/hello.go",
				"bla-blub.go.go",
				"1/2/yo.go.go",
			},
			fileSrcGlobPath: "**/*.go",
		},

		{
			files: []string{
				"base.go",
				"1/yo.go",
				"1/2/3/nonono.no",
				"1/2/3/three.go",
				"1/2/two.go",
			},
			dir: "1/2/3",
			expectedMatches: []string{
				"base.go",
				"1/yo.go",
				"1/2/3/nonono.no",
				"1/2/3/three.go",
				"1/2/two.go",
			},
			fileSrcGlobPath: "**",
		},

		{
			files: []string{
				"base.go",
				"1/yo.go",
				"1/2/3/nonono.no",

				"1/2/3/three.go",
				"1/2/two.go",
			},
			dir: "1/2/3",
			expectedMatches: []string{
				"1/yo.go",
				"1/2/3/three.go",
				"1/2/two.go",
			},
			fileSrcGlobPath: "1/**/*.go",
		},
	}

	for _, tc := range testcases {
		tempdir, cleanupFunc := fstest.CreateTempDir(t)
		defer cleanupFunc()

		if len(tc.dir) != 0 {
			err := os.MkdirAll(filepath.Join(tempdir, tc.dir), os.ModePerm)
			if err != nil {
				t.Fatal("creating subdirectories failed:", err)
			}
		}

		createFiles(t, tempdir, tc.files)

		fs := NewResolver(filepath.Join(tempdir, tc.fileSrcGlobPath))
		resolvedFiles, err := fs.Resolve()
		if err != nil {
			t.Fatal("resolving glob path:", err)
		}

		checkFilesInResolvedFiles(t, tempdir, resolvedFiles, tc)
	}

}
