package baur

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v1/internal/testutils/fstest"
	"github.com/simplesurance/baur/v1/internal/testutils/gittest"
	"github.com/simplesurance/baur/v1/pkg/cfg"
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

			r := NewCachingInputResolver()

			for _, f := range tc.filesToCreate {
				fstest.WriteToFile(t, []byte(f), filepath.Join(tempDir, f))
			}

			if strings.Contains(tc.name, "git") {
				gittest.CreateRepository(t, tempDir)
				if len(tc.filesToCreate) > 0 {
					gittest.CommitFilesToGit(t, tempDir)
				}
			}

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
