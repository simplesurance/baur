package baur

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/simplesurance/baur/testutils/fstest"
	"github.com/simplesurance/baur/testutils/strtest"
)

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

func checkFilesInResolvedFiles(t *testing.T, tempdir string, resolvedBuildInput []BuildInput, tc *testcase) {
	resolvedFiles := []*File{}
	for _, i := range resolvedBuildInput {
		resolvedFiles = append(resolvedFiles, i.(*File))
	}

	if len(resolvedFiles) != len(tc.expectedMatches) {
		t.Errorf("resolved to %d files (%v), expected %d (%+v)",
			len(resolvedFiles), resolvedFiles,
			len(tc.expectedMatches), tc)
	}

	for _, e := range resolvedFiles {
		if !strtest.InSlice(tc.expectedMatches, e.RelPath()) {
			t.Errorf("%q (%q) was returned but is not in expected return slice (%+v), testcase: %+v",
				e, e.RelPath(), tc.expectedMatches, tc)
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
		&testcase{
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

		&testcase{
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

		&testcase{
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

		&testcase{
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

		&testcase{
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

		fs := NewFileGlobPath(tempdir, tc.fileSrcGlobPath)
		resolvedFiles, err := fs.Resolve()
		if err != nil {
			t.Fatal("resolving glob path:", err)
		}

		checkFilesInResolvedFiles(t, tempdir, resolvedFiles, tc)
	}

}
