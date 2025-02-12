//go:build dbtest

package postgres

import (
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v5/pkg/storage"
)

func TestDelete(t *testing.T) {
	startTime := time.Now().Add(-1 * time.Minute)

	tr := storage.TaskRunFull{
		TaskRun: storage.TaskRun{
			ApplicationName:  "baurHimself",
			TaskName:         "build",
			VCSRevision:      "1",
			VCSIsDirty:       false,
			StartTimestamp:   startTime,
			StopTimestamp:    time.Now(),
			Result:           storage.ResultSuccess,
			TotalInputDigest: "1234567890",
		},
		Inputs: storage.Inputs{
			Files: []*storage.InputFile{
				{
					Path:   "main.go",
					Digest: "45",
				},
				{
					Path:   "abc.go",
					Digest: "01",
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
						URI:                  "abc",
						UploadStartTimestamp: time.Now(),
						UploadStopTimestamp:  time.Now().Add(5 * time.Second),
						Method:               storage.UploadMethodS3,
					},
					{
						URI:                  "efg",
						UploadStartTimestamp: time.Now(),
						UploadStopTimestamp:  time.Now().Add(5 * time.Second),
						Method:               storage.UploadMethodS3,
					},
				},
			},
		},
	}

	clt, cleanupFn := newTestClient(t)
	defer cleanupFn()

	require.NoError(t, clt.Init(ctx))

	_, err := clt.SaveTaskRun(ctx, &tr)
	require.NoError(t, err)

	checkResultFn := func(result *storage.TaskRunsDeleteResult) {
		assert.Equal(t, int64(1), result.DeletedVCS)
		assert.Equal(t, int64(1), result.DeletedApps)
		assert.Equal(t, int64(1), result.DeletedTasks)
		assert.Equal(t, int64(2), result.DeletedInputs)
		assert.Equal(t, int64(1), result.DeletedOutputs)
		assert.Equal(t, int64(2), result.DeletedUploads)
	}

	result, err := clt.TaskRunsDelete(ctx, startTime.Add(time.Minute), true)
	require.NoError(t, err)
	checkResultFn(result)

	result, err = clt.TaskRunsDelete(ctx, startTime.Add(time.Minute), false)
	require.NoError(t, err)
	checkResultFn(result)

	require.NotNil(t, result)

	tableNames := allTableNames(t, clt)
	for _, tableName := range tableNames {
		if tableName == "migrations" {
			continue
		}
		assert.Truef(
			t,
			tableIsEmpty(t, clt, tableName),
			"table %s is not empty", tableName,
		)
	}
}

func tableIsEmpty(t *testing.T, clt *Client, tableName string) bool {
	var result bool
	q := fmt.Sprintf(`SELECT EXISTS (SELECT * FROM %s LIMIT 1)`, pgx.Identifier{tableName}.Sanitize())

	err := clt.db.QueryRow(t.Context(), q).Scan(&result)
	require.NoErrorf(t, err, "checking if table is empty failed, query: %q", q)
	return !result
}

func allTableNames(t *testing.T, clt *Client) []string {
	var result []string

	const q = `SELECT tablename
	       FROM pg_catalog.pg_tables
	      WHERE  schemaname = 'public'
	      `

	rows, err := clt.db.Query(t.Context(), q)
	require.NoError(t, err, "querying table names failed")
	for rows.Next() {
		var tableName string
		require.NoError(t, rows.Scan(&tableName), "scanning table name failed")
		result = append(result, tableName)
	}

	require.NoError(t, rows.Err(), "iterating over table name rows failed")
	return result
}
