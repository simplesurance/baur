//go:build dbtest

package postgres

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v5/pkg/storage"
)

type testRunOpt func(*storage.TaskRunFull)

func withOutputs(o ...*storage.Output) testRunOpt {
	return func(r *storage.TaskRunFull) { r.Outputs = o }
}

func withInputs(in storage.Inputs) testRunOpt {
	return func(r *storage.TaskRunFull) { r.Inputs = in }
}

func withRevision(rev string, dirty bool) testRunOpt {
	return func(r *storage.TaskRunFull) {
		r.VCSRevision = rev
		r.VCSIsDirty = dirty
	}
}

func withResult(res storage.Result) testRunOpt {
	return func(r *storage.TaskRunFull) { r.Result = res }
}

// saveTestRun stores a task run that started at start and ran for 1 minute
// and returns its ID.
func saveTestRun(
	t *testing.T,
	clt *Client,
	app, task, inputDigest string,
	start time.Time,
	opts ...testRunOpt,
) int {
	t.Helper()

	run := &storage.TaskRunFull{
		TaskRun: storage.TaskRun{
			ApplicationName:  app,
			TaskName:         task,
			VCSRevision:      "rev",
			StartTimestamp:   start,
			StopTimestamp:    start.Add(time.Minute),
			Result:           storage.ResultSuccess,
			TotalInputDigest: inputDigest,
		},
	}

	for _, o := range opts {
		o(run)
	}

	id, err := clt.SaveTaskRun(ctx, run)
	require.NoError(t, err)

	return id
}

func fileOutput(name, digest string, uploads ...*storage.Upload) *storage.Output {
	return &storage.Output{
		Name:      name,
		Type:      storage.ArtifactTypeFile,
		Digest:    digest,
		SizeBytes: 10,
		Uploads:   uploads,
	}
}

func upload(method storage.UploadMethod, uri string) *storage.Upload {
	now := time.Now()

	return &storage.Upload{
		URI:                  uri,
		UploadStartTimestamp: now,
		UploadStopTimestamp:  now.Add(time.Second),
		Method:               method,
	}
}

func TestReleaseLifecycle(t *testing.T) {
	clt, cleanup := newTestClient(t)
	defer cleanup()
	require.NoError(t, clt.Init(ctx))

	now := time.Now().Truncate(time.Second)

	buildID := saveTestRun(t, clt, "app", "build", "d1", now, withOutputs(
		fileOutput("bin", "o1",
			upload(storage.UploadMethodS3, "s3://bucket/key"),
			upload(storage.UploadMethodFileCopy, "/dest/bin"),
		),
	))
	testID := saveTestRun(t, clt, "app", "test", "d2", now)
	firstAppID := saveTestRun(t, clt, "aaa", "z", "d3", now)

	require.NoError(t, clt.CreateRelease(ctx, "v1", now, []int{buildID, testID, firstAppID}, strings.NewReader("meta")))

	t.Run("ReleaseExists", func(t *testing.T) {
		ok, err := clt.ReleaseExists(ctx, "v1")
		require.NoError(t, err)
		assert.True(t, ok)

		ok, err = clt.ReleaseExists(ctx, "v2")
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("ReleaseTaskRuns lists every upload, ordered by app, task, output, uri", func(t *testing.T) {
		got, err := clt.ReleaseTaskRuns(ctx, "v1")
		require.NoError(t, err)
		require.Len(t, got, 4)

		// sorted by app name first
		assert.Equal(t, &storage.ReleaseTaskRunsResult{
			AppName: "aaa", TaskName: "z", RunID: firstAppID,
		}, got[0])
		got = got[1:]

		// "/dest/bin" sorts before "s3://bucket/key"
		assert.Equal(t, "app", got[0].AppName)
		assert.Equal(t, "build", got[0].TaskName)
		assert.Equal(t, buildID, got[0].RunID)
		assert.Equal(t, "bin", got[0].OutputName)
		assert.NotZero(t, got[0].OutputID)
		assert.Equal(t, "/dest/bin", got[0].URI)
		assert.Equal(t, storage.UploadMethodFileCopy, got[0].UploadMethod)

		assert.Equal(t, "build", got[1].TaskName)
		assert.Equal(t, got[0].OutputID, got[1].OutputID)
		assert.Equal(t, "s3://bucket/key", got[1].URI)
		assert.Equal(t, storage.UploadMethodS3, got[1].UploadMethod)

		// task run without output: only ids and names are set
		assert.Equal(t, &storage.ReleaseTaskRunsResult{
			AppName: "app", TaskName: "test", RunID: testID,
		}, got[2])
	})

	t.Run("ReleaseMetadata", func(t *testing.T) {
		got, err := clt.ReleaseMetadata(ctx, "v1")
		require.NoError(t, err)
		assert.Equal(t, []byte("meta"), got)
	})

	t.Run("unknown release returns ErrNotExist", func(t *testing.T) {
		_, err := clt.ReleaseTaskRuns(ctx, "nope")
		require.ErrorIs(t, err, storage.ErrNotExist)

		_, err = clt.ReleaseMetadata(ctx, "nope")
		require.ErrorIs(t, err, storage.ErrNotExist)
	})

	t.Run("duplicate release name returns ErrExists", func(t *testing.T) {
		err := clt.CreateRelease(ctx, "v1", now, []int{buildID}, nil)
		require.ErrorIs(t, err, storage.ErrExists)
	})

	t.Run("release without task runs is rejected and not created", func(t *testing.T) {
		err := clt.CreateRelease(ctx, "empty", now, nil, nil)
		require.Error(t, err)

		ok, err := clt.ReleaseExists(ctx, "empty")
		require.NoError(t, err)
		assert.False(t, ok, "failed CreateRelease must roll back the release row")
	})

	t.Run("release without metadata returns empty metadata", func(t *testing.T) {
		require.NoError(t, clt.CreateRelease(ctx, "nometa", now, []int{testID}, nil))

		got, err := clt.ReleaseMetadata(ctx, "nometa")
		require.NoError(t, err)
		assert.Empty(t, got)
	})
}

func TestReleasesDelete(t *testing.T) {
	clt, cleanup := newTestClient(t)
	defer cleanup()
	require.NoError(t, clt.Init(ctx))

	now := time.Now().Truncate(time.Second)
	runID := saveTestRun(t, clt, "app", "build", "d1", now)

	require.NoError(t, clt.CreateRelease(ctx, "old", now.Add(-48*time.Hour), []int{runID}, nil))
	require.NoError(t, clt.CreateRelease(ctx, "new", now, []int{runID}, nil))

	cutoff := now.Add(-24 * time.Hour)

	exists := func(name string) bool {
		ok, err := clt.ReleaseExists(ctx, name)
		require.NoError(t, err)

		return ok
	}

	t.Run("pretend reports but does not delete", func(t *testing.T) {
		res, err := clt.ReleasesDelete(ctx, cutoff, true)
		require.NoError(t, err)
		assert.Equal(t, int64(1), res.DeletedReleases)

		assert.True(t, exists("old"))
		assert.True(t, exists("new"))
	})

	t.Run("deletes only releases older than the cutoff", func(t *testing.T) {
		res, err := clt.ReleasesDelete(ctx, cutoff, false)
		require.NoError(t, err)
		assert.Equal(t, int64(1), res.DeletedReleases)

		assert.False(t, exists("old"))
		assert.True(t, exists("new"))

		_, err = clt.ReleaseTaskRuns(ctx, "old")
		require.ErrorIs(t, err, storage.ErrNotExist)
	})

	t.Run("nothing left to delete", func(t *testing.T) {
		res, err := clt.ReleasesDelete(ctx, cutoff, false)
		require.NoError(t, err)
		assert.Zero(t, res.DeletedReleases)
	})
}

func TestTaskRunsDeleteKeepsNewRunsAndRunsOfReleases(t *testing.T) {
	clt, cleanup := newTestClient(t)
	defer cleanup()
	require.NoError(t, clt.Init(ctx))

	now := time.Now().Truncate(time.Second)

	oldID := saveTestRun(t, clt, "app", "old", "d-old", now.Add(-72*time.Hour))
	releasedID := saveTestRun(t, clt, "app", "released", "d-rel", now.Add(-72*time.Hour))
	newID := saveTestRun(t, clt, "app", "new", "d-new", now)

	require.NoError(t, clt.CreateRelease(ctx, "v1", now, []int{releasedID}, nil))

	cutoff := now.Add(-24 * time.Hour)

	res, err := clt.TaskRunsDelete(ctx, cutoff, true)
	require.NoError(t, err)
	assert.Equal(t, int64(1), res.DeletedTaskRuns)

	for _, id := range []int{oldID, releasedID, newID} {
		_, err := clt.TaskRun(ctx, id)
		require.NoError(t, err, "pretend must not delete task run %d", id)
	}

	res, err = clt.TaskRunsDelete(ctx, cutoff, false)
	require.NoError(t, err)
	assert.Equal(t, int64(1), res.DeletedTaskRuns)
	assert.Equal(t, int64(1), res.DeletedTasks, "only the task of the deleted run is unused")
	assert.Zero(t, res.DeletedApps, "app is still used by the remaining runs")

	_, err = clt.TaskRun(ctx, oldID)
	require.ErrorIs(t, err, storage.ErrNotExist)

	_, err = clt.TaskRun(ctx, releasedID)
	require.NoError(t, err, "task runs that are part of a release must not be deleted")

	_, err = clt.TaskRun(ctx, newID)
	require.NoError(t, err, "task runs newer than the cutoff must not be deleted")
}
