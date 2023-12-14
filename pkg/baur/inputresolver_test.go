package baur

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v3/internal/digest"
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
			log.RedirectToTestingLog(t)

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
			r := NewInputResolver(vcsState, tempDir)

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

	log.RedirectToTestingLog(t)

	tempDir := t.TempDir()

	vcsState, err := vcs.GetState(tempDir, log.Debugf)
	require.NoError(t, err)
	r := NewInputResolver(vcsState, tempDir)

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

	oldExecDebugFfN := exec.DefaultLogFn
	exec.DefaultLogFn = t.Logf
	t.Cleanup(func() {
		exec.DefaultLogFn = oldExecDebugFfN
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
	r := NewInputResolver(vcsState, gitDir)

	resolvedFiles, err := r.resolveFileInputs(appDir, []cfg.FileInputs{
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
			log.RedirectToTestingLog(t)

			for k, v := range tc.EnvVars {
				t.Setenv(k, v)
			}

			resolver := NewInputResolver(&vcs.NoVCsState{}, ".")
			resolver.setEnvVars()
			resolvedEnvVars, err := resolver.resolveEnvVarInputs(tc.Inputs)
			if tc.ExpectedErrStr != "" {
				require.ErrorContains(t, err, tc.ExpectedErrStr)
			}

			require.EqualValues(t, tc.ExpectedResolvedEnvVars, resolvedEnvVars)
		})
	}

}

func TestExcludedFiles(t *testing.T) {
	testcases := []struct {
		Name           string
		FilesToCreate  []string
		Inputs         cfg.Input
		ExpectedResult []string
	}{
		{
			Name:          "basic",
			FilesToCreate: []string{"abc", "main.c", "readme.md", "changelog.md"},
			Inputs: cfg.Input{
				Files: []cfg.FileInputs{
					{
						Paths: []string{"*"},
					},
				},
				ExcludedFiles: cfg.FileExcludeList{
					Paths: []string{"*.md", "abc"},
				},
			},
			ExpectedResult: []string{"main.c"},
		},

		{
			Name:          "exclude_does_not_match",
			FilesToCreate: []string{"abc"},
			Inputs: cfg.Input{
				Files: []cfg.FileInputs{
					{
						Paths: []string{"abc"},
					},
				},
				ExcludedFiles: cfg.FileExcludeList{
					Paths: []string{"xyz", "*.md"},
				},
			},
			ExpectedResult: []string{"abc"},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			log.RedirectToTestingLog(t)
			tempDir := t.TempDir()

			for _, f := range tc.FilesToCreate {
				fstest.WriteToFile(t, []byte(f), filepath.Join(tempDir, f))
			}

			resolver := NewInputResolver(&vcs.NoVCsState{}, tempDir)
			result, err := resolver.Resolve(context.Background(), tempDir, &Task{
				Directory:        tempDir,
				UnresolvedInputs: &tc.Inputs,
			})

			require.NoError(t, err)

			strResult := toStrSlice(result)
			assert.ElementsMatch(t, strResult, tc.ExpectedResult)
		})
	}
}

func toStrSlice[S ~[]T, T fmt.Stringer](in S) []string {
	result := make([]string, len(in))

	for i, e := range in {
		result[i] = e.String()
	}

	return result
}

type goSourceResolverMock struct {
	result []string
}

func (g *goSourceResolverMock) Resolve(
	_ context.Context,
	_ string,
	_ []string,
	_ []string,
	_ bool,
	_ []string,
) ([]string, error) {
	return g.result, nil
}

func TestGoResolverFilesAreExcluded(t *testing.T) {
	log.RedirectToTestingLog(t)
	baseDir := filepath.FromSlash("/tmp")

	resolver := NewInputResolver(&vcs.NoVCsState{}, baseDir)
	resolver.goSourceResolver = &goSourceResolverMock{
		result: []string{filepath.Join(baseDir, "main.go"), filepath.Join(baseDir, "atm", "atm.go")},
	}

	result, err := resolver.Resolve(
		context.Background(),
		filepath.FromSlash("/tmp"),
		&Task{
			UnresolvedInputs: &cfg.Input{
				GolangSources: []cfg.GolangSources{{}},
				ExcludedFiles: cfg.FileExcludeList{
					Paths: []string{"main.go"},
				},
			},
			Directory: baseDir,
		},
	)
	require.NoError(t, err)

	strResult := toStrSlice(result)
	assert.ElementsMatch(t, strResult, []string{filepath.Join("atm", "atm.go")})
}

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
				require.ErrorIs(t, err, os.ErrNotExist)
				require.Empty(t, result)
			},
		},
		{
			testdir:   "file_broken",
			inputPath: "symlink",
			validateFn: func(t *testing.T, err error, result []Input) {
				require.ErrorIs(t, err, os.ErrNotExist)
				require.Empty(t, result)
			},
		},
		{
			testdir:   "file_broken",
			inputPath: "**",
			validateFn: func(t *testing.T, err error, result []Input) {
				require.ErrorIs(t, err, os.ErrNotExist)
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
			validateFn: func(t *testing.T, err error, _ []Input) {
				require.ErrorContains(t, err, "file does not exist")
			},
		},
		{
			testdir:   "symlinks",
			inputPath: "directory_containing_broken_symlin**/**",
			validateFn: func(t *testing.T, err error, _ []Input) {
				require.ErrorIs(t, err, os.ErrNotExist)
			},
		},
	}

	for _, tc := range testcases {
		t.Run(fmt.Sprintf("%s/%s", tc.testdir, tc.inputPath), func(t *testing.T) {
			log.RedirectToTestingLog(t)

			testDir := filepath.Join(testdataDir, "symlinks", tc.testdir)

			vcsState, err := vcs.GetState(testDir, log.Debugf)
			require.NoError(t, err)
			r := NewInputResolver(vcsState, tc.testdir)

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

type gitFileTC struct {
	CreateGitRepo        bool
	AddToGitBeforeChange bool
	AddToGitAfterChange  bool
}

func newGitFileTcVariations() []*gitFileTC {
	return []*gitFileTC{
		{
			CreateGitRepo:        false,
			AddToGitBeforeChange: false,
			AddToGitAfterChange:  false,
		},
		{
			CreateGitRepo:        true,
			AddToGitBeforeChange: false,
			AddToGitAfterChange:  false,
		},
		{
			CreateGitRepo:        true,
			AddToGitBeforeChange: true,
			AddToGitAfterChange:  false,
		},
		{
			CreateGitRepo:        true,
			AddToGitBeforeChange: true,
			AddToGitAfterChange:  true,
		},
		{
			CreateGitRepo:        true,
			AddToGitBeforeChange: false,
			AddToGitAfterChange:  true,
		},
	}
}

func TestFileContentChanges(t *testing.T) {
	for _, tc := range newGitFileTcVariations() {
		t.Run(
			fmt.Sprintf("gitrepo:%+v,commitbeforechange:%v,commitafterchange:%v",
				tc.CreateGitRepo, tc.AddToGitBeforeChange, tc.AddToGitAfterChange),
			func(t *testing.T) {
				log.RedirectToTestingLog(t)
				tempDir := t.TempDir()
				if tc.CreateGitRepo {
					gittest.CreateRepository(t, tempDir)
				}

				const fRel = "file 🙀 very\\ sp€cイal"
				f := filepath.Join(tempDir, fRel)
				fstest.WriteToFile(t, []byte("111"), f)

				if tc.AddToGitBeforeChange {
					gittest.CommitFilesToGit(t, tempDir)
				}

				task := Task{
					RepositoryRoot: tempDir,
					Directory:      tempDir,
					UnresolvedInputs: &cfg.Input{
						Files: []cfg.FileInputs{{Paths: []string{fRel}}},
					},
				}
				before := resolveInputs(t, &task)
				fstest.WriteToFile(t, []byte("11113"), f)
				if tc.AddToGitAfterChange {
					gittest.CommitFilesToGit(t, tempDir)
				}

				after := resolveInputs(t, &task)
				require.NotEqual(t, before.String(), after.String())
			})
	}
}

type symlinkTestInfo struct {
	TempDir               string
	SymlinkPath           string
	SymlinkTargetFilePath string
	Task                  *Task
	TotalInputDigest      *digest.Digest
}

func prepareSymlinkTestDir(t *testing.T, createGitRepo, commitFilesToGit bool) *symlinkTestInfo {
	t.Helper()
	tempDir := t.TempDir()
	if createGitRepo {
		gittest.CreateRepository(t, tempDir)
	}
	f := filepath.Join(tempDir, "file")

	const symlinkRel = "slink"
	symlink := filepath.Join(tempDir, symlinkRel)

	fstest.WriteToFile(t, []byte("hello"), f)
	require.NoError(t, os.Symlink(f, symlink))

	if commitFilesToGit {
		gittest.CommitFilesToGit(t, tempDir)
	}

	task := Task{
		RepositoryRoot: tempDir,
		Directory:      tempDir,
		UnresolvedInputs: &cfg.Input{
			Files: []cfg.FileInputs{{Paths: []string{symlinkRel}}},
		},
	}

	digest := resolveInputs(t, &task)

	return &symlinkTestInfo{
		TempDir:               tempDir,
		SymlinkPath:           symlink,
		SymlinkTargetFilePath: f,
		Task:                  &task,
		TotalInputDigest:      digest,
	}
}

func resolveInputs(t *testing.T, task *Task) *digest.Digest {
	t.Helper()

	vcsState, err := vcs.GetState(task.RepositoryRoot, t.Logf)
	require.NoError(t, err)

	resolver := NewInputResolver(vcsState, task.RepositoryRoot)
	result, err := resolver.Resolve(
		context.Background(),
		task.RepositoryRoot,
		task,
	)
	require.NoError(t, err)

	digest, err := NewInputs(result).Digest()
	require.NoError(t, err)
	require.NotEmpty(t, digest.String())
	return digest
}

func TestSymlinkIsReplacedByTargetFile(t *testing.T) {
	t.Skip("fails because of bug: https://github.com/simplesurance/baur/issues/490")
	log.RedirectToTestingLog(t)

	info := prepareSymlinkTestDir(t, false, false)

	require.NoError(t, os.Remove(info.SymlinkPath))
	require.NoError(t, os.Rename(info.SymlinkTargetFilePath, info.SymlinkPath))

	digestAfterReplace := resolveInputs(t, info.Task)
	require.NotEqual(t, info.TotalInputDigest.String(), digestAfterReplace.String())
}

func TestSymlinkTargetPathChangesFileIsSame(t *testing.T) {
	t.Skip("fails because of bug: https://github.com/simplesurance/baur/issues/491")
	log.RedirectToTestingLog(t)

	info := prepareSymlinkTestDir(t, false, false)

	newTargetFilePath := filepath.Join(info.TempDir, "file1")

	require.NoError(t, os.Rename(info.SymlinkTargetFilePath, newTargetFilePath))
	require.NoError(t, os.Remove(info.SymlinkPath))
	require.NoError(t, os.Symlink(newTargetFilePath, info.SymlinkPath))

	digestAfter := resolveInputs(t, info.Task)
	require.NotEqual(t, info.TotalInputDigest.String(), digestAfter.String())
}

func TestSymlinkTargetFileContentChanges(t *testing.T) {
	for _, tc := range newGitFileTcVariations() {
		t.Run(
			fmt.Sprintf("gitrepo:%+v,commitbeforechange:%v,commitafterchange:%v",
				tc.CreateGitRepo, tc.AddToGitBeforeChange, tc.AddToGitAfterChange),
			func(t *testing.T) {
				log.RedirectToTestingLog(t)
				info := prepareSymlinkTestDir(t, tc.CreateGitRepo, tc.AddToGitBeforeChange)
				fstest.WriteToFile(t, []byte("hello symlink"), info.SymlinkTargetFilePath)
				if tc.AddToGitAfterChange {
					gittest.CommitFilesToGit(t, info.TempDir)
				}
				digestAfter := resolveInputs(t, info.Task)
				require.NotEqual(t, info.TotalInputDigest.String(), digestAfter.String())
			})
	}
}

func TestSymlinkTargetFilePermissionsChange(t *testing.T) {
	t.Skip("fails because of bug: https://github.com/simplesurance/baur/issues/492")
	log.RedirectToTestingLog(t)

	info := prepareSymlinkTestDir(t, false, false)
	require.NoError(t, os.Chmod(info.SymlinkTargetFilePath, 0777))

	digestAfter := resolveInputs(t, info.Task)
	require.NotEqual(t, info.TotalInputDigest.String(), digestAfter.String())
}
