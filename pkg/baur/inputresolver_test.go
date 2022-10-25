package baur

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v3/internal/exec"
	"github.com/simplesurance/baur/v3/internal/fs"
	"github.com/simplesurance/baur/v3/internal/log"
	"github.com/simplesurance/baur/v3/internal/testutils/fstest"
	"github.com/simplesurance/baur/v3/internal/testutils/gittest"
	"github.com/simplesurance/baur/v3/internal/vcs"
	"github.com/simplesurance/baur/v3/pkg/cfg"
)

func TestFilesOptional(t *testing.T) {
	testcases := []struct {
		name          string
		filesToCreate []string
		task          Task
		expectError   bool
	}{
		{
			name:          "file_input_optional_missing_2defs",
			filesToCreate: []string{"file.1"},
			task: Task{
				UnresolvedInputs: &cfg.Input{
					Files: []cfg.FileInputs{
						{
							Paths:    []string{"*.1"},
							Optional: false,
						},
						{
							Paths:    []string{"*.2"},
							Optional: true,
						},
					},
				},
			},
		},
		{
			name:          "gitfile_input_optional_missing_2defs",
			filesToCreate: []string{"file.1"},
			task: Task{
				UnresolvedInputs: &cfg.Input{
					Files: []cfg.FileInputs{
						{
							Paths:          []string{"*.1"},
							Optional:       false,
							GitTrackedOnly: true,
						},
						{
							Paths:          []string{"*.2"},
							Optional:       true,
							GitTrackedOnly: true,
						},
					},
				},
			},
		},

		{
			name:          "file_input_optional_missing",
			filesToCreate: []string{"file.1"},
			task: Task{
				UnresolvedInputs: &cfg.Input{
					Files: []cfg.FileInputs{
						{
							Paths:    []string{"*.1", "*.2"},
							Optional: true,
						},
					},
				},
			},
		},
		{
			name:          "gitfile_input_optional_missing",
			filesToCreate: []string{"file.1"},
			task: Task{
				UnresolvedInputs: &cfg.Input{
					Files: []cfg.FileInputs{
						{
							Paths:          []string{"*.1", "*.2"},
							Optional:       true,
							GitTrackedOnly: true,
						},
					},
				},
			},
		},

		{
			name:          "file_input_optional_exists",
			filesToCreate: []string{"file.1", "file.2"},
			task: Task{
				UnresolvedInputs: &cfg.Input{
					Files: []cfg.FileInputs{
						{
							Paths:    []string{"*.1", "*.2"},
							Optional: true,
						},
					},
				},
			},
		},
		{
			name:          "gitfile_input_optional_exists",
			filesToCreate: []string{"file.1", "file.2"},
			task: Task{
				UnresolvedInputs: &cfg.Input{
					Files: []cfg.FileInputs{
						{
							Paths:          []string{"*.1", "*.2"},
							Optional:       true,
							GitTrackedOnly: true,
						},
					},
				},
			},
		},

		{
			name:          "file_input_optional_2defs_one_missing",
			filesToCreate: []string{"file.1"},
			expectError:   true,
			task: Task{
				UnresolvedInputs: &cfg.Input{
					Files: []cfg.FileInputs{
						{
							Paths:    []string{"*.1", "*.2"},
							Optional: false,
						},
					},
				},
			},
		},
		{
			name:          "gitfile_input_optional_2defs_one_missing",
			filesToCreate: []string{"file.1"},
			expectError:   true,
			task: Task{
				UnresolvedInputs: &cfg.Input{
					Files: []cfg.FileInputs{
						{
							Paths:          []string{"*.1", "*.2"},
							Optional:       false,
							GitTrackedOnly: true,
						},
					},
				},
			},
		},

		{
			name:          "file_input_exists",
			filesToCreate: []string{"file.1", "file.2"},
			task: Task{
				UnresolvedInputs: &cfg.Input{
					Files: []cfg.FileInputs{
						{
							Paths:    []string{"*.1", "*.2"},
							Optional: false,
						},
					},
				},
			},
		},
		{
			name:          "gitfile_input_exists",
			filesToCreate: []string{"file.1", "file.2"},
			task: Task{
				UnresolvedInputs: &cfg.Input{
					Files: []cfg.FileInputs{
						{
							Paths:          []string{"*.1", "*.2"},
							Optional:       false,
							GitTrackedOnly: true,
						},
					},
				},
			},
		},

		{
			name:          "file_input_missing",
			filesToCreate: []string{"file.1"},
			expectError:   true,
			task: Task{
				UnresolvedInputs: &cfg.Input{
					Files: []cfg.FileInputs{
						{
							Paths:    []string{"*.1", "*.2"},
							Optional: false,
						},
					},
				},
			},
		},
		{
			name:          "gitfile_input_missing",
			filesToCreate: []string{"file.1"},
			expectError:   true,
			task: Task{
				UnresolvedInputs: &cfg.Input{
					Files: []cfg.FileInputs{
						{
							Paths:          []string{"*.1", "*.2"},
							Optional:       false,
							GitTrackedOnly: true,
						},
					},
				},
			},
		},

		{
			name:        "optional_dir_not_exist",
			expectError: false,
			task: Task{
				UnresolvedInputs: &cfg.Input{
					Files: []cfg.FileInputs{
						{
							Paths:          []string{"dir/**"},
							Optional:       true,
							GitTrackedOnly: false,
						},
					},
				},
			},
		},
		{
			name:        "gitfile_optional_dir_not_exist",
			expectError: false,
			task: Task{
				UnresolvedInputs: &cfg.Input{
					Files: []cfg.FileInputs{
						{
							Paths:          []string{"dir/**"},
							Optional:       true,
							GitTrackedOnly: true,
						},
					},
				},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()

			for _, f := range tc.filesToCreate {
				fstest.WriteToFile(t, []byte(f), filepath.Join(tempDir, f))
			}

			if strings.Contains(tc.name, "git") {
				gittest.CreateRepository(t, tempDir)
				if len(tc.filesToCreate) > 0 {
					gittest.CommitFilesToGit(t, tempDir)
				}
			}

			vcsState, err := vcs.GetState(tempDir, log.Debugf)
			require.NoError(t, err)
			r := NewInputResolver(vcsState)

			tc.task.Directory = tempDir

			result, err := r.Resolve(context.Background(), tempDir, &tc.task)
			if tc.expectError {
				require.Error(t, err)
				require.Empty(t, result)

				return
			}

			require.NoError(t, err)

			t.Logf("result: %+v", result)

			for _, f := range tc.filesToCreate {
				var found bool

				for _, in := range result {
					if in.String() == f {
						found = true
						break
					}

				}

				assert.True(t, found, "%s missing in result", f)
			}
		})
	}
}

func TestPathsAfterMissingOptionalOneAreNotIgnored(t *testing.T) {
	const fname = "hello"

	tempDir := t.TempDir()

	vcsState, err := vcs.GetState(tempDir, log.Debugf)
	require.NoError(t, err)
	r := NewInputResolver(vcsState)

	fstest.WriteToFile(t, []byte("123"), filepath.Join(tempDir, fname))

	result, err := r.Resolve(context.Background(), tempDir, &Task{
		Directory: tempDir,
		UnresolvedInputs: &cfg.Input{
			Files: []cfg.FileInputs{
				{
					Paths:    []string{"doesnotexist", fname},
					Optional: true,
				},
			},
		},
	})

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, fname, result[0].String())
}

func TestResolverIgnoredGitUntrackedFiles(t *testing.T) {
	log.RedirectToTestingLog(t)

	oldExecDebugFfN := exec.DefaultDebugfFn
	exec.DefaultDebugfFn = t.Logf
	t.Cleanup(func() {
		exec.DefaultDebugfFn = oldExecDebugFfN
	})

	gitDir := t.TempDir()
	gitDir, err := fs.RealPath(gitDir)
	require.NoError(t, err)

	appDir := filepath.Join(gitDir, "subdir")
	gittest.CreateRepository(t, gitDir)

	const trackedFilename = "file1.txt"
	fstest.WriteToFile(t, []byte("123"), filepath.Join(appDir, trackedFilename))
	gittest.CommitFilesToGit(t, gitDir)

	// file2.txt is untracked
	const untrackedFilename = "file2.txt"
	fstest.WriteToFile(t, []byte("123"), filepath.Join(appDir, untrackedFilename))

	vcsState, err := vcs.GetState(gitDir, log.Debugf)
	require.NoError(t, err)
	r := NewInputResolver(vcsState)

	resolvedFiles, err := r.resolveFileInputs(gitDir, appDir, []cfg.FileInputs{
		{
			Paths:          []string{"**"},
			GitTrackedOnly: true,
			Optional:       false,
		},
	})

	require.NoError(t, err)
	require.NotEmpty(t, resolvedFiles)
	assert.ElementsMatch(t, []string{filepath.Join(appDir, trackedFilename)}, resolvedFiles)
}

func TestResolveEnvVarInputs(t *testing.T) {
	testcases := []struct {
		Name                    string
		EnvVars                 map[string]string
		Inputs                  []cfg.EnvVarsInputs
		ExpectedErrStr          string
		ExpectedResolvedEnvVars map[string]string
	}{
		{
			Name: "prefix_glob",
			EnvVars: map[string]string{
				"VarA":            "testval",
				"VarB":            "blubb",
				"Var_XYYZ":        "XYZ",
				"BLUBB":           "3",
				"ANOTHER_ONEXY":   "",
				"VAR_NOT_MATCHED": "9",
			},
			Inputs: []cfg.EnvVarsInputs{
				{
					Names: []string{
						"Var*",
						"B?UB?",
						"*ONEXY",
					},
				},
			},
			ExpectedResolvedEnvVars: map[string]string{
				"VarA":          "testval",
				"VarB":          "blubb",
				"Var_XYYZ":      "XYZ",
				"BLUBB":         "3",
				"ANOTHER_ONEXY": "",
			},
		},
		{
			Name: "noglobs",
			EnvVars: map[string]string{
				"VarA":  "testval",
				"VAR_B": "blubb",
			},
			Inputs: []cfg.EnvVarsInputs{
				{
					Names: []string{"VarA", "VAR_B"},
				},
			},
			ExpectedResolvedEnvVars: map[string]string{
				"VarA":  "testval",
				"VAR_B": "blubb",
			},
		},

		{
			Name: "missing_optional_succeeds",
			EnvVars: map[string]string{
				"VarA": "testval",
			},
			Inputs: []cfg.EnvVarsInputs{
				{
					Names: []string{"VarA"},
				},
				{
					Names:    []string{"VarB"},
					Optional: true,
				},
			},
			ExpectedResolvedEnvVars: map[string]string{
				"VarA": "testval",
			},
		},

		{
			Name: "missing_var_fails",
			EnvVars: map[string]string{
				"VarA": "testval",
			},
			Inputs: []cfg.EnvVarsInputs{
				{
					Names: []string{"VarA", "VAR_B"},
				},
			},
			ExpectedErrStr: "environment variable \"VAR_B\" is undefined",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			for k, v := range tc.EnvVars {
				t.Setenv(k, v)
			}

			resolver := NewInputResolver(&vcs.NoVCsState{})
			resolver.setEnvVars()
			resolvedEnvVars, err := resolver.resolveEnvVarInputs(tc.Inputs)
			if tc.ExpectedErrStr != "" {
				require.ErrorContains(t, err, tc.ExpectedErrStr)
			}

			require.EqualValues(t, tc.ExpectedResolvedEnvVars, resolvedEnvVars)
		})
	}

}
