//go:build linux || darwin

package baur

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v2/internal/log"
	"github.com/simplesurance/baur/v2/internal/vcs"
	"github.com/simplesurance/baur/v2/pkg/cfg"
)

var testdataDir string

func init() {
	_, testfile, _, ok := runtime.Caller(0)
	if !ok {
		panic("could not get test filename")
	}

	absPath, err := filepath.Abs(testfile)
	if err != nil {
		panic(fmt.Sprintf(
			" could not get absolute path of testfile (%s): %s",
			testfile, err))
	}
	fmt.Println(absPath)

	testdataDir = filepath.Join(filepath.Dir(absPath), "testdata")
}

func TestResolveSymlink(t *testing.T) {
	log.RedirectToTestingLog(t)

	testcases := []struct {
		testdir    string
		inputPath  string
		validateFn func(t *testing.T, err error, result []Input)
	}{
		{
			testdir:   "directory_broken",
			inputPath: "symlink",
			validateFn: func(t *testing.T, err error, result []Input) {
				require.ErrorContains(t, err, "no such file or directory")
				require.Len(t, result, 0)
			},
		},
		{
			testdir:   "file_broken",
			inputPath: "symlink",
			validateFn: func(t *testing.T, err error, result []Input) {
				require.ErrorContains(t, err, "no such file or directory")
				require.Len(t, result, 0)
			},
		},
		{
			testdir:   "file",
			inputPath: "symlink",
			validateFn: func(t *testing.T, err error, result []Input) {
				require.NoError(t, err)
				require.Len(t, result, 1)

				fileInput, ok := result[0].(*InputFile)
				if !ok {
					t.Fatalf("result[0] has type %t, expected *Inputfile", result[0])
				}

				assert.Equal(t, "symlink", fileInput.RelPath())
			},
		},
		{
			testdir:   "directory_with_files",
			inputPath: "**",
			validateFn: func(t *testing.T, err error, result []Input) {
				require.NoError(t, err)
				require.Len(t, result, 1)

				fileInput, ok := result[0].(*InputFile)
				if !ok {
					t.Fatalf("result[0] has type %t, expected *Inputfile", result[0])
				}

				assert.Equal(t, filepath.Join("thedirectory", "arealfile"), fileInput.RelPath())
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.testdir, func(t *testing.T) {
			log.RedirectToTestingLog(t)

			testDir := filepath.Join(testdataDir, "symlinks", tc.testdir)

			vcsState, err := vcs.GetState(testDir, log.Debugf)
			require.NoError(t, err)
			r := NewInputResolver(vcsState)

			result, err := r.Resolve(context.Background(), testDir, &Task{
				Directory: testDir,
				UnresolvedInputs: &cfg.Input{Files: []cfg.FileInputs{{
					Paths:    []string{tc.inputPath},
					Optional: false,
				}}},
			})

			tc.validateFn(t, err, result)
		})
	}
}
