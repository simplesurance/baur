package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v5/internal/testutils/fstest"
	"github.com/simplesurance/baur/v5/internal/testutils/strtest"
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

type testcase struct {
	files           []string
	dir             string
	expectedMatches []string
	fileSrcGlobPath string
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
		tempdir := fstest.TempDir(t)

		// The path separators in the test cases are Unix style "/", they need to be converted to "\" when running on Windows
		for i := range tc.expectedMatches {
			tc.expectedMatches[i] = filepath.FromSlash(tc.expectedMatches[i])
		}

		if len(tc.dir) != 0 {
			err := os.MkdirAll(filepath.Join(tempdir, tc.dir), os.ModePerm)
			if err != nil {
				t.Fatal("creating subdirectories failed:", err)
			}
		}

		createFiles(t, tempdir, tc.files)

		resolvedFiles, err := FileGlob(filepath.Join(tempdir, tc.fileSrcGlobPath))
		if err != nil {
			t.Fatal("resolving glob path:", err)
		}

		checkFilesInResolvedFiles(t, tempdir, resolvedFiles, tc)
	}
}

func TestGlobMatch(t *testing.T) {
	tcs := []struct {
		pattern     string
		path        string
		expectMatch bool
	}{
		{
			pattern:     "?",
			path:        "a",
			expectMatch: true,
		},
	}

	for _, tc := range tcs {
		t.Run(fmt.Sprintf("pattern:%s,path:%s", tc.pattern, tc.path), func(t *testing.T) {
			match, err := MatchGlob(tc.pattern, tc.path)
			require.NoError(t, err)
			assert.Equal(t, tc.expectMatch, match)
		})
	}
}
