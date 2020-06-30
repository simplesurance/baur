// +build integrationtest

package postgres

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/storage"
)

// drop the local monotonic values from timestamps and rounding it is required
// to prevent that comparsions of local and retrieved objects fail because of the monotonic clock value or
// minor timestamp changes
// See also: https://github.com/stretchr/testify/issues/502
func taskRunDropMonotonicTimevals(tr *storage.TaskRun) *storage.TaskRun {
	tr.StartTimestamp = tr.StartTimestamp.Round(time.Millisecond)
	tr.StopTimestamp = tr.StopTimestamp.Round(time.Millisecond)

	return tr
}

func outputDropMonotonicTimevals(outputs []*storage.Output) []*storage.Output {
	for _, o := range outputs {
		for _, upload := range o.Uploads {
			upload.UploadStartTimestamp = upload.UploadStartTimestamp.Round(time.Millisecond)
			upload.UploadStopTimestamp = upload.UploadStopTimestamp.Round(time.Millisecond)
		}
	}

	return outputs
}

func TestLatestTaskRunByDigest(t *testing.T) {
	client, cleanupFn := newTestClient(t)
	defer cleanupFn()

	require.NoError(t, client.Init(ctx))

	run1 := storage.TaskRunFull{
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
		Inputs: []*storage.Input{
			{
				URI:    "main.go",
				Digest: "45",
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
	}

	run2 := run1
	run2.StopTimestamp = run2.StopTimestamp.Add(time.Second)

	_, err := client.SaveTaskRun(ctx, &run1)
	require.NoError(t, err)

	id, err := client.SaveTaskRun(ctx, &run2)
	require.NoError(t, err)

	latestTaskRun, err := client.LatestTaskRunByDigest(ctx, run2.ApplicationName, run2.TaskName, run2.TotalInputDigest)
	require.NoError(t, err)

	assert.Equal(t, id, latestTaskRun.ID, "wrong record id")
	assert.Equal(t, taskRunDropMonotonicTimevals(&run2.TaskRun), taskRunDropMonotonicTimevals(&latestTaskRun.TaskRun))
}

func TestLatestTaskRunByDigest_ReturnsErrNotExist(t *testing.T) {
	client, cleanupFn := newTestClient(t)
	defer cleanupFn()

	require.NoError(t, client.Init(ctx))
	taskRun, err := client.LatestTaskRunByDigest(ctx, "myapp", "mytask", "241abc")

	assert.Equal(t, storage.ErrNotExist, err)
	assert.Nil(t, taskRun)
}

func TestTaskRun_ReturnsErrNotExist(t *testing.T) {
	client, cleanupFn := newTestClient(t)
	defer cleanupFn()

	require.NoError(t, client.Init(ctx))
	taskRun, err := client.TaskRun(ctx, 113124)

	assert.Equal(t, storage.ErrNotExist, err)
	assert.Nil(t, taskRun)
}

func TestInputs_ReturnsErrNotExist(t *testing.T) {
	client, cleanupFn := newTestClient(t)
	defer cleanupFn()

	require.NoError(t, client.Init(ctx))
	inputs, err := client.Inputs(ctx, 113124)

	assert.Equal(t, storage.ErrNotExist, err)
	assert.Nil(t, inputs)
}

func TestOutputs_ReturnsErrNotExist(t *testing.T) {
	client, cleanupFn := newTestClient(t)
	defer cleanupFn()

	require.NoError(t, client.Init(ctx))
	outputs, err := client.Outputs(ctx, 113124)

	assert.Equal(t, storage.ErrNotExist, err)
	assert.Nil(t, outputs)
}

// TODO: Write Tests for:
// - Outputs()
// - Inputs()

func TestOutputs(t *testing.T) {
	client, cleanupFn := newTestClient(t)
	defer cleanupFn()

	require.NoError(t, client.Init(ctx))

	run := storage.TaskRunFull{
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
		Inputs: []*storage.Input{
			{
				URI:    "main.go",
				Digest: "45",
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
			{
				Name:      "binary2",
				Type:      storage.ArtifactTypeFile,
				Digest:    "4561",
				SizeBytes: 2,
				Uploads: []*storage.Upload{
					{
						URI:                  "s3://tmp/test",
						UploadStartTimestamp: time.Now(),
						UploadStopTimestamp:  time.Now().Add(5 * time.Second),
						Method:               storage.UploadMethodS3,
					},
					{
						URI:                  "file://myftp.com/file.bin",
						UploadStartTimestamp: time.Now(),
						UploadStopTimestamp:  time.Now().Add(5 * time.Second),
						Method:               storage.UploadMethodFileCopy,
					},
				},
			},
		},
	}

	id, err := client.SaveTaskRun(ctx, &run)
	require.NoError(t, err)
	assert.Greater(t, id, 0)

	outputs, err := client.Outputs(ctx, id)
	require.NoError(t, err)

	assert.ElementsMatch(t, outputDropMonotonicTimevals(run.Outputs), outputDropMonotonicTimevals(outputs))
}

func TestInputs(t *testing.T) {
	client, cleanupFn := newTestClient(t)
	defer cleanupFn()

	require.NoError(t, client.Init(ctx))

	run := storage.TaskRunFull{
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
		Inputs: []*storage.Input{
			{
				URI:    "main.go",
				Digest: "45",
			},

			{
				URI:    "util.go",
				Digest: "46",
			},
			{
				URI:    "file://Makefile",
				Digest: "47",
			},
		},
	}

	id, err := client.SaveTaskRun(ctx, &run)
	require.NoError(t, err)
	assert.Greater(t, id, 0)

	inputs, err := client.Inputs(ctx, id)
	require.NoError(t, err)

	assert.Equal(t, run.Inputs, inputs)
}

func TestTaskRun(t *testing.T) {
	client, cleanupFn := newTestClient(t)
	defer cleanupFn()

	require.NoError(t, client.Init(ctx))

	run := storage.TaskRunFull{
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
		Inputs: []*storage.Input{
			{
				URI:    "main.go",
				Digest: "45",
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
			{
				Name:      "binary2",
				Type:      storage.ArtifactTypeFile,
				Digest:    "4561",
				SizeBytes: 2,
				Uploads: []*storage.Upload{
					{
						URI:                  "s3://tmp/test",
						UploadStartTimestamp: time.Now(),
						UploadStopTimestamp:  time.Now().Add(5 * time.Second),
						Method:               storage.UploadMethodS3,
					},
					{
						URI:                  "file://myftp.com/file.bin",
						UploadStartTimestamp: time.Now(),
						UploadStopTimestamp:  time.Now().Add(5 * time.Second),
						Method:               storage.UploadMethodFileCopy,
					},
				},
			},
		},
	}

	id, err := client.SaveTaskRun(ctx, &run)
	require.NoError(t, err)
	assert.Greater(t, id, 0)

	taskRun, err := client.TaskRun(ctx, id)
	require.NoError(t, err)
	assert.NotNil(t, taskRun)

	assert.Equal(t, taskRunDropMonotonicTimevals(&run.TaskRun), taskRunDropMonotonicTimevals(&taskRun.TaskRun))
}

func TestTaskRuns(t *testing.T) {
	client, cleanupFn := newTestClient(t)
	defer cleanupFn()

	require.NoError(t, client.Init(ctx))

	run := storage.TaskRunFull{
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
		Inputs: []*storage.Input{
			{
				URI:    "main.go",
				Digest: "45",
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
			{
				Name:      "binary2",
				Type:      storage.ArtifactTypeFile,
				Digest:    "4561",
				SizeBytes: 2,
				Uploads: []*storage.Upload{
					{
						URI:                  "s3://tmp/test",
						UploadStartTimestamp: time.Now(),
						UploadStopTimestamp:  time.Now().Add(5 * time.Second),
						Method:               storage.UploadMethodS3,
					},
					{
						URI:                  "file://myftp.com/file.bin",
						UploadStartTimestamp: time.Now(),
						UploadStopTimestamp:  time.Now().Add(5 * time.Second),
						Method:               storage.UploadMethodFileCopy,
					},
				},
			},
		},
	}

	taskRunDropMonotonicTimevals(&run.TaskRun)

	id, err := client.SaveTaskRun(ctx, &run)
	require.NoError(t, err)
	assert.Greater(t, id, 0)

	run1 := run
	run1.StartTimestamp = run1.StartTimestamp.Add(time.Second)
	run1.TaskName = "check"
	taskRunDropMonotonicTimevals(&run1.TaskRun)

	id1, err := client.SaveTaskRun(ctx, &run1)
	require.NoError(t, err)
	assert.Greater(t, id1, 0)
	assert.NotEqual(t, id, id1)

	testcases := []*struct {
		name    string
		filters []*storage.Filter
		sorters []*storage.Sorter

		expectedTaskRuns []*storage.TaskRunWithID
		expectedError    error
	}{
		{
			name: "EqAppNameAndTaskName",
			filters: []*storage.Filter{
				&storage.Filter{
					Field:    storage.FieldTaskName,
					Operator: storage.OpEQ,
					Value:    run1.TaskName,
				},

				&storage.Filter{
					Field:    storage.FieldApplicationName,
					Operator: storage.OpEQ,
					Value:    run1.ApplicationName,
				},
			},
			expectedTaskRuns: []*storage.TaskRunWithID{
				&storage.TaskRunWithID{
					ID:      id1,
					TaskRun: run1.TaskRun,
				},
			},
		},

		{
			name: "INAppNames",
			filters: []*storage.Filter{
				&storage.Filter{
					Field:    storage.FieldApplicationName,
					Operator: storage.OpIN,
					Value:    []string{run1.ApplicationName, "testApp"},
				},
			},
			expectedTaskRuns: []*storage.TaskRunWithID{
				&storage.TaskRunWithID{
					ID:      id,
					TaskRun: run.TaskRun,
				},

				&storage.TaskRunWithID{
					ID:      id1,
					TaskRun: run1.TaskRun,
				},
			},
		},

		{
			name: "appNameOrderByDurationAsc",
			filters: []*storage.Filter{
				&storage.Filter{
					Field:    storage.FieldApplicationName,
					Operator: storage.OpEQ,
					Value:    run.ApplicationName,
				},
			},
			sorters: []*storage.Sorter{
				&storage.Sorter{
					storage.FieldDuration,
					storage.OrderAsc,
				},
			},
			expectedTaskRuns: []*storage.TaskRunWithID{
				&storage.TaskRunWithID{
					ID:      id,
					TaskRun: run.TaskRun,
				},
				&storage.TaskRunWithID{
					ID:      id1,
					TaskRun: run1.TaskRun,
				},
			},
		},

		{
			name: "appNameOrderByDurationDesc",
			filters: []*storage.Filter{
				&storage.Filter{
					Field:    storage.FieldApplicationName,
					Operator: storage.OpEQ,
					Value:    run.ApplicationName,
				},
			},
			sorters: []*storage.Sorter{
				&storage.Sorter{
					storage.FieldDuration,
					storage.OrderDesc,
				},
			},
			expectedTaskRuns: []*storage.TaskRunWithID{
				&storage.TaskRunWithID{
					ID:      id1,
					TaskRun: run1.TaskRun,
				},

				&storage.TaskRunWithID{
					ID:      id,
					TaskRun: run.TaskRun,
				},
			},
		},

		{
			name: "NoMatch",
			filters: []*storage.Filter{
				&storage.Filter{
					Field:    storage.FieldID,
					Operator: storage.OpEQ,
					Value:    -500,
				},
			},
			expectedError: storage.ErrNotExist,
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			var result []*storage.TaskRunWithID

			err := client.TaskRuns(ctx, testcase.filters, testcase.sorters, func(tr *storage.TaskRunWithID) error {
				result = append(result, tr)
				return nil
			})
			assert.Equal(t, testcase.expectedError, err)

			for _, taskRun := range result {
				taskRunDropMonotonicTimevals(&taskRun.TaskRun)
			}

			assert.ElementsMatch(t, testcase.expectedTaskRuns, result)
		})
	}

}

func TestTaskRunQueryRunWithoutOutputWithoutVCS(t *testing.T) {
	client, cleanupFn := newTestClient(t)
	defer cleanupFn()

	require.NoError(t, client.Init(ctx))

	run := storage.TaskRunFull{
		TaskRun: storage.TaskRun{
			ApplicationName:  "baurHimself",
			TaskName:         "build",
			StartTimestamp:   time.Now(),
			StopTimestamp:    time.Now().Add(5 * time.Minute),
			Result:           storage.ResultSuccess,
			TotalInputDigest: "1234567890",
		},
		Inputs: []*storage.Input{
			{
				URI:    "main.go",
				Digest: "45",
			},
		},
	}

	taskRunDropMonotonicTimevals(&run.TaskRun)

	id, err := client.SaveTaskRun(ctx, &run)
	require.NoError(t, err)
	assert.Greater(t, id, 0)

	tr, err := client.TaskRun(ctx, id)
	require.NoError(t, err)
	require.NotNil(t, tr)

	assert.Equal(t, run.TaskRun, tr.TaskRun)
	assert.Equal(t, id, tr.ID)
}
