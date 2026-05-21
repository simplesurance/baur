//go:build dbtest

package postgres

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v5/pkg/storage"
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
					assert.NoError(t, err)   //nolint: testifylint
					assert.Greater(t, id, 0) //nolint: testifylint

					return
				}

				require.Error(t, err)
			}
		})
	}
}

func TestSaveTaskRunWithBigIntInputFileID(t *testing.T) {
	client, cleanupFn := newTestClient(t)
	defer cleanupFn()

	require.NoError(t, client.Init(ctx))
	const maxBigInt int64 = 1<<63 - 1
	_, err := client.db.Exec(ctx, "SELECT setval('input_file_id_seq', $1, true)", maxBigInt-1)
	require.NoError(t, err)

	_, err = client.SaveTaskRun(ctx, &storage.TaskRunFull{
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
	})
	require.NoError(t, err)

	var inputFileID, inputFileFK int64
	err = client.db.QueryRow(ctx, `
		SELECT input_file.id, task_run_file_input.input_file_id
		  FROM input_file
		  JOIN task_run_file_input ON input_file.id = task_run_file_input.input_file_id
		 WHERE input_file.path = 'main.go'
		   AND input_file.digest = '45'
	`).Scan(&inputFileID, &inputFileFK)
	require.NoError(t, err)
	require.Equal(t, maxBigInt, inputFileID)
	require.Equal(t, maxBigInt, inputFileFK)
}
