package baur

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v3/internal/log"
	"github.com/simplesurance/baur/v3/internal/vcs"
	"github.com/simplesurance/baur/v3/pkg/cfg"
)

func relPathsFromInputs(t *testing.T, in []Input) []string {
	res := make([]string, len(in))

	for i, r := range in {
		fi, ok := r.(*InputFile)
		if !ok {
			t.Fatalf("result[%d] has type %t, expected *Inputfile", i, r)
		}
		res[i] = fi.RelPath()
	}

	return res
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
				require.ErrorContains(t, err, "file does not exist")
				require.Empty(t, result)
			},
		},
		{
			testdir:   "directory_broken",
			inputPath: "**",
			validateFn: func(t *testing.T, err error, result []Input) {
				require.ErrorContains(t, err, "no such file or directory")
				require.Empty(t, result)
			},
		},
		{
			testdir:   "file_broken",
			inputPath: "symlink",
			validateFn: func(t *testing.T, err error, result []Input) {
				require.ErrorContains(t, err, "file does not exist")
				require.Empty(t, result)
			},
		},
		{
			testdir:   "file_broken",
			inputPath: "**",
			validateFn: func(t *testing.T, err error, result []Input) {
				require.ErrorContains(t, err, "no such file or directory")
				require.Empty(t, result)
			},
		},
		{
			testdir:   "file",
			inputPath: "symlink",
			validateFn: func(t *testing.T, err error, result []Input) {
				require.NoError(t, err)
				assert.ElementsMatch(t,
					[]string{"symlink"},
					relPathsFromInputs(t, result),
				)
			},
		},
		{
			testdir:   "file",
			inputPath: "**",
			validateFn: func(t *testing.T, err error, result []Input) {
				require.NoError(t, err)
				assert.ElementsMatch(t,
					[]string{"thefile", "symlink"},
					relPathsFromInputs(t, result),
				)
			},
		},
		{
			testdir:   "directory_with_files",
			inputPath: "**",
			validateFn: func(t *testing.T, err error, result []Input) {
				require.NoError(t, err)
				assert.ElementsMatch(t,
					[]string{
						filepath.Join("symlink", "arealfile"),
						filepath.Join("thedirectory", "arealfile"),
					},
					relPathsFromInputs(t, result),
				)
			},
		},
		{
			testdir:   "symlinks/directory_containing_broken_symlink",
			inputPath: "**",
			validateFn: func(t *testing.T, err error, result []Input) {
				require.ErrorContains(t, err, "file does not exist")
			},
		},
		{
			testdir:   "symlinks",
			inputPath: "directory_containing_broken_symlin**/**",
			validateFn: func(t *testing.T, err error, result []Input) {
				t.Log(err)
				require.ErrorContains(t, err, "file does not exist")
			},
		},
	}

	for _, tc := range testcases {
		t.Run(fmt.Sprintf("%s/%s", tc.testdir, tc.inputPath), func(t *testing.T) {
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
