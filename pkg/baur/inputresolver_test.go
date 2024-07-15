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

	"github.com/simplesurance/baur/v5/internal/digest"
	"github.com/simplesurance/baur/v5/internal/exec"
	"github.com/simplesurance/baur/v5/internal/fs"
	"github.com/simplesurance/baur/v5/internal/log"
	"github.com/simplesurance/baur/v5/internal/testutils/fstest"
	"github.com/simplesurance/baur/v5/internal/testutils/gittest"
	"github.com/simplesurance/baur/v5/internal/vcs/git"
	"github.com/simplesurance/baur/v5/pkg/cfg"
)

type DummyGitUntrackedFilesResolver struct{}

func (*DummyGitUntrackedFilesResolver) WithoutUntracked(...string) ([]string, error) {
	return nil, nil
}

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

			gittest.CreateRepository(t, tempDir)
			if strings.Contains(tc.name, "git") && len(tc.filesToCreate) > 0 {
				gittest.CommitFilesToGit(t, tempDir)
			}

			r := NewInputResolver(git.NewRepository(tempDir), tempDir, nil, true)

			tc.task.Directory = tempDir

			result, err := r.Resolve(context.Background(), &tc.task)
			if tc.expectError {
				require.Error(t, err)
				require.Empty(t, result)

				return
			}

			require.NoError(t, err)

			t.Logf("result: %+v", result)

			for _, f := range tc.filesToCreate {
				var found bool

				for _, in := range result.Inputs() {
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
	gittest.CreateRepository(t, tempDir)

	r := NewInputResolver(git.NewRepository(tempDir), tempDir, nil, true)

	fstest.WriteToFile(t, []byte("123"), filepath.Join(tempDir, fname))

	result, err := r.Resolve(context.Background(), &Task{
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
	require.Len(t, result.Inputs(), 1)
	assert.Equal(t, fname, result.inputs[0].String())
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

	r := NewInputResolver(git.NewRepository(gitDir), gitDir, nil, true)

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

			resolver := NewInputResolver(&DummyGitUntrackedFilesResolver{}, ".", nil, true)
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
			gittest.CreateRepository(t, tempDir)

			for _, f := range tc.FilesToCreate {
				fstest.WriteToFile(t, []byte(f), filepath.Join(tempDir, f))
			}

			resolver := NewInputResolver(git.NewRepository(tempDir), tempDir, nil, true)
			result, err := resolver.Resolve(context.Background(), &Task{
				Directory:        tempDir,
				UnresolvedInputs: &tc.Inputs,
			})

			require.NoError(t, err)

			strResult := toStrSlice(result.Inputs())
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
	baseDir := fstest.TempDir(t)
	f1 := filepath.Join(baseDir, "main.go")
	f2 := filepath.Join(baseDir, "atm", "atm.go")
	fstest.WriteToFile(t, []byte("a"), f1)
	fstest.WriteToFile(t, []byte("b"), f2)
	gittest.CreateRepository(t, baseDir)

	resolver := NewInputResolver(git.NewRepository(baseDir), baseDir, nil, true)
	resolver.goSourceResolver = &goSourceResolverMock{result: []string{f1, f2}}

	result, err := resolver.Resolve(
		context.Background(),
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

	strResult := toStrSlice(result.Inputs())
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
		validateFn func(t *testing.T, err error, result *Inputs)
	}{
		{
			testdir:   "directory_broken",
			inputPath: "symlink",
			validateFn: func(t *testing.T, err error, result *Inputs) {
				require.ErrorContains(t, err, "file does not exist")
				require.Nil(t, result)
			},
		},
		{
			testdir:   "directory_broken",
			inputPath: "**",
			validateFn: func(t *testing.T, err error, result *Inputs) {
				require.ErrorIs(t, err, os.ErrNotExist)
				require.Nil(t, result)
			},
		},
		{
			testdir:   "file_broken",
			inputPath: "symlink",
			validateFn: func(t *testing.T, err error, result *Inputs) {
				require.ErrorIs(t, err, os.ErrNotExist)
				require.Nil(t, result)
			},
		},
		{
			testdir:   "file_broken",
			inputPath: "**",
			validateFn: func(t *testing.T, err error, result *Inputs) {
				require.ErrorIs(t, err, os.ErrNotExist)
				require.Nil(t, result)
			},
		},
		{
			testdir:   "file",
			inputPath: "symlink",
			validateFn: func(t *testing.T, err error, result *Inputs) {
				require.NoError(t, err)
				assert.ElementsMatch(t,
					[]string{filepath.Join("file", "symlink")},
					relPathsFromInputs(t, result.Inputs()),
				)
			},
		},
		{
			testdir:   "file",
			inputPath: "**",
			validateFn: func(t *testing.T, err error, result *Inputs) {
				require.NoError(t, err)
				assert.ElementsMatch(t,
					[]string{
						filepath.Join("file", "thefile"),
						filepath.Join("file", "symlink"),
					},
					relPathsFromInputs(t, result.Inputs()),
				)
			},
		},
		{
			testdir:   "directory_with_files",
			inputPath: "**",
			validateFn: func(t *testing.T, err error, result *Inputs) {
				require.NoError(t, err)
				assert.ElementsMatch(t,
					[]string{
						filepath.Join("directory_with_files", "thedirectory", "arealfile"),
						filepath.Join("directory_with_files", "symlink", "arealfile"),
					},
					relPathsFromInputs(t, result.Inputs()),
				)
			},
		},
		{
			testdir:   "symlinks/directory_containing_broken_symlink",
			inputPath: "**",
			validateFn: func(t *testing.T, err error, _ *Inputs) {
				require.ErrorContains(t, err, "file does not exist")
			},
		},
		{
			testdir:   "symlinks",
			inputPath: "directory_containing_broken_symlin**/**",
			validateFn: func(t *testing.T, err error, _ *Inputs) {
				require.ErrorIs(t, err, os.ErrNotExist)
			},
		},
	}

	for _, tc := range testcases {
		t.Run(fmt.Sprintf("%s/%s", tc.testdir, tc.inputPath), func(t *testing.T) {
			log.RedirectToTestingLog(t)

			repoDir := filepath.Join(testdataDir, "symlinks")

			gittest.CreateRepository(t, repoDir)
			r := NewInputResolver(git.NewRepository(repoDir), repoDir, nil, true)

			result, err := r.Resolve(context.Background(), &Task{
				Directory: filepath.Join(repoDir, tc.testdir),
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
	AddToGitBeforeChange bool
	AddToGitAfterChange  bool
}

func newGitFileTcVariations() []*gitFileTC {
	return []*gitFileTC{
		{
			AddToGitBeforeChange: false,
			AddToGitAfterChange:  false,
		},
		{
			AddToGitBeforeChange: true,
			AddToGitAfterChange:  false,
		},
		{
			AddToGitBeforeChange: true,
			AddToGitAfterChange:  true,
		},
		{
			AddToGitBeforeChange: false,
			AddToGitAfterChange:  true,
		},
	}
}

func TestFileContentChanges(t *testing.T) {
	for _, tc := range newGitFileTcVariations() {
		t.Run(
			fmt.Sprintf("commitbeforechange:%v,commitafterchange:%v",
				tc.AddToGitBeforeChange, tc.AddToGitAfterChange),
			func(t *testing.T) {
				log.RedirectToTestingLog(t)
				tempDir := t.TempDir()
				gittest.CreateRepository(t, tempDir)

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
				_, before := resolveInputs(t, &task, !tc.AddToGitBeforeChange)
				fstest.WriteToFile(t, []byte("11113"), f)
				if tc.AddToGitAfterChange {
					gittest.CommitFilesToGit(t, tempDir)
				}

				_, after := resolveInputs(t, &task, !tc.AddToGitAfterChange)
				require.NotEqual(t, before.String(), after.String())
			})
	}
}

type symlinkTestInfo struct {
	TempDir                  string
	SymlinkPath              string
	SymlinkTargetFilePath    string
	SymlinkTargetFileRelPath string
	Task                     *Task
	TotalInputDigest         *digest.Digest
	ResolvedInputs           *Inputs
}

func prepareSymlinkTestDir(t *testing.T, commitFilesToGit bool) *symlinkTestInfo {
	t.Helper()
	tempDir := fstest.TempDir(t)
	gittest.CreateRepository(t, tempDir)
	const fRel = "file"
	f := filepath.Join(tempDir, fRel)

	const symlinkRel = "slink"
	symlink := filepath.Join(tempDir, symlinkRel)

	fstest.WriteToFile(t, []byte("hello"), f)
	fstest.Symlink(t, f, symlink)

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

	inputs, digest := resolveInputs(t, &task, !commitFilesToGit)

	return &symlinkTestInfo{
		TempDir:                  tempDir,
		SymlinkPath:              symlink,
		SymlinkTargetFilePath:    f,
		SymlinkTargetFileRelPath: fRel,
		Task:                     &task,
		TotalInputDigest:         digest,
		ResolvedInputs:           inputs,
	}
}

func resolveInputs(t *testing.T, task *Task, hashGitUntracked bool) (*Inputs, *digest.Digest) {
	t.Helper()

	resolver := NewInputResolver(git.NewRepository(task.RepositoryRoot), task.RepositoryRoot, nil, hashGitUntracked)
	result, err := resolver.Resolve(
		context.Background(),
		task,
	)
	require.NoError(t, err)
	digest, err := result.Digest()
	require.NoError(t, err)
	require.NotEmpty(t, digest.String())
	return result, digest
}

func TestSymlinkIsReplacedByTargetFile(t *testing.T) {
	log.RedirectToTestingLog(t)

	info := prepareSymlinkTestDir(t, false)

	require.NoError(t, os.Remove(info.SymlinkPath))
	require.NoError(t, os.Rename(info.SymlinkTargetFilePath, info.SymlinkPath))

	_, digestAfterReplace := resolveInputs(t, info.Task, true)
	require.NotEqual(t, info.TotalInputDigest.String(), digestAfterReplace.String())
}

func TestSymlinkTargetPathChangesFileIsSame(t *testing.T) {
	log.RedirectToTestingLog(t)

	info := prepareSymlinkTestDir(t, false)

	newTargetFilePath := filepath.Join(info.TempDir, "file1")

	require.NoError(t, os.Rename(info.SymlinkTargetFilePath, newTargetFilePath))
	require.NoError(t, os.Remove(info.SymlinkPath))
	fstest.Symlink(t, newTargetFilePath, info.SymlinkPath)

	_, digestAfter := resolveInputs(t, info.Task, true)
	require.NotEqual(t, info.TotalInputDigest.String(), digestAfter.String())
}

func TestSymlinkTargetFileContentChanges(t *testing.T) {
	for _, tc := range newGitFileTcVariations() {
		t.Run(
			fmt.Sprintf("commitbeforechange:%v,commitafterchange:%v",
				tc.AddToGitBeforeChange, tc.AddToGitAfterChange),
			func(t *testing.T) {
				log.RedirectToTestingLog(t)
				info := prepareSymlinkTestDir(t, tc.AddToGitBeforeChange)
				fstest.WriteToFile(t, []byte("hello symlink"), info.SymlinkTargetFilePath)
				if tc.AddToGitAfterChange {
					gittest.CommitFilesToGit(t, info.TempDir)
				}
				inputs, digestAfter := resolveInputs(t, info.Task, !tc.AddToGitAfterChange)
				require.NotEqual(t, info.TotalInputDigest.String(), digestAfter.String())

				assert.Len(t, inputs.inputs, 1)
				inf, ok := inputs.inputs[0].(*InputFile)
				require.True(t, ok)
				assert.Equal(t, info.SymlinkPath, inf.absPath)
				assert.Equal(t, info.SymlinkTargetFileRelPath, inf.repoRelRealPath, "unexpected symlink rel path")
			})
	}
}

func TestHashGitUntrackedFilesDisabled(t *testing.T) {
	const fname = "hello"

	log.RedirectToTestingLog(t)
	tempDir := t.TempDir()
	gittest.CreateRepository(t, tempDir)

	fstest.WriteToFile(t, []byte("123"), filepath.Join(tempDir, fname))

	task := &Task{
		Directory: tempDir,
		UnresolvedInputs: &cfg.Input{
			Files: []cfg.FileInputs{{Paths: []string{fname}}},
		},
	}

	r := NewInputResolver(git.NewRepository(tempDir), tempDir, nil, false)
	_, err := r.Resolve(context.Background(), task)
	require.ErrorIs(t, err, git.ErrObjectNotFound)
}

func TestFileInSymlinkDir(t *testing.T) {
	const fname = "hello"
	const subdir = "realDir"
	const dirSlink = "dirSymlink"

	testFn := func(hashGitUntracked bool) {
		t.Run(fmt.Sprintf("hashGitUntracked:%t", hashGitUntracked), func(t *testing.T) {
			log.RedirectToTestingLog(t)
			tempDir := fstest.TempDir(t)

			f := filepath.Join(tempDir, subdir, fname)
			fstest.WriteToFile(t, []byte("123"), f)
			slink := filepath.Join(tempDir, dirSlink)
			fstest.Symlink(t, filepath.Join(tempDir, subdir), slink)

			gittest.CreateRepository(t, tempDir)
			if hashGitUntracked {
				var err error

				gittest.CommitFilesToGit(t, tempDir)
				require.NoError(t, err)
			}

			task := &Task{
				Directory: tempDir,
				UnresolvedInputs: &cfg.Input{
					Files: []cfg.FileInputs{{Paths: []string{filepath.Join(dirSlink, fname)}}},
				},
			}

			r := NewInputResolver(git.NewRepository(tempDir), tempDir, nil, !hashGitUntracked)
			inputs, err := r.Resolve(context.Background(), task)
			require.NoError(t, err)
			require.Len(t, inputs.Inputs(), 1)
			inf, ok := inputs.Inputs()[0].(*InputFile)
			require.True(t, ok)
			assert.Equal(t, filepath.Join(slink, fname), inf.absPath)
			assert.Equal(t, filepath.Join(dirSlink, fname), inf.repoRelPath)
			assert.Equal(t, filepath.Join(subdir, fname), inf.repoRelRealPath, "unexpected real path")

			_, err = inputs.Digest()
			require.NoError(t, err)
		})
	}

	testFn(true)
	testFn(false)
}
