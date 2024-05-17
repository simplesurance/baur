//go:build dbtest
// +build dbtest

package postgres

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v3/pkg/storage"
)

func TestSaveTaskRun(t *testing.T) {
	testcases := []*struct {
		name string

		taskRuns      []*storage.TaskRunFull
		expectSuccess []bool
	}{
		{
			name: "1",
			taskRuns: []*storage.TaskRunFull{
				{
					TaskRun: storage.TaskRun{
						ApplicationName:  "baurHimself",
						TaskName:         "build",
						VCSRevision:      "1",
						VCSIsDirty:       false,
						StartTimestamp:   time.Now(),
						StopTimestamp:    time.Now().Add(5 * time.Minute),
						Result:           storage.ResultSuccess,
						TotalInputDigest: "1234567890",
					},
					Inputs: storage.Inputs{
						Files: []*storage.InputFile{
							{
								Path:   "main.go",
								Digest: "45",
							},
						},
					},
					Outputs: []*storage.Output{
						{
							Name:      "binary",
							Type:      storage.ArtifactTypeFile,
							Digest:    "456",
							SizeBytes: 300,
							Uploads: []*storage.Upload{
								{
									UploadStartTimestamp: time.Now(),
									UploadStopTimestamp:  time.Now().Add(5 * time.Second),
									Method:               storage.UploadMethodS3,
								},
							},
						},
					},
				},
			},
			expectSuccess: []bool{true},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			if len(tc.taskRuns) != len(tc.expectSuccess) {
				t.Fatal("taskRuns and expectSuccess slice of testcase do not contain same number of elements")
			}

			client, cleanupFn := newTestClient(t)
			defer cleanupFn()

			require.NoError(t, client.Init(ctx))

			for i := range tc.taskRuns {
				taskRun := tc.taskRuns[i]
				expectedResult := tc.expectSuccess[i]

				id, err := client.SaveTaskRun(ctx, taskRun)

				if expectedResult {
					assert.NoError(t, err) //nolint: testifylint
					assert.Greater(t, id, 0)

					return
				}

				require.Error(t, err)
			}
		})
	}
}
