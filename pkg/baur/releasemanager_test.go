package baur

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v5/internal/set"
	"github.com/simplesurance/baur/v5/pkg/storage"
)

type fakeReleaseStorer struct {
	storage.Storer

	taskRuns    []*storage.ReleaseTaskRunsResult
	taskRunsErr error
	metadata    []byte
	metadataErr error
}

func (f *fakeReleaseStorer) ReleaseTaskRuns(context.Context, string) ([]*storage.ReleaseTaskRunsResult, error) {
	return f.taskRuns, f.taskRunsErr
}

func (f *fakeReleaseStorer) ReleaseMetadata(context.Context, string) ([]byte, error) {
	return f.metadata, f.metadataErr
}

type download struct{ bucket, key, dest string }

type fakeS3Downloader struct {
	downloads []download
	err       error
}

func (f *fakeS3Downloader) Download(_ context.Context, bucket, key, dest string) error {
	if f.err != nil {
		return f.err
	}

	f.downloads = append(f.downloads, download{bucket, key, dest})

	return nil
}

type nopLogger struct{}

func (nopLogger) Debugf(string, ...any) {}

func s3Run(app, task string, runID, outputID int, name, uri string) *storage.ReleaseTaskRunsResult {
	return &storage.ReleaseTaskRunsResult{
		AppName: app, TaskName: task, RunID: runID, OutputID: outputID,
		OutputName: name, URI: uri, UploadMethod: storage.UploadMethodS3,
	}
}

func newTestReleaseManager(st *fakeReleaseStorer, s3 *fakeS3Downloader) *ReleaseManager {
	return &ReleaseManager{storage: st, s3clt: s3, logger: nopLogger{}}
}

func TestDownloadOutputs(t *testing.T) {
	ctx := context.Background()

	t.Run("downloads each S3 output into <dest>/<taskID>/<basename>", func(t *testing.T) {
		dest := t.TempDir()
		s3 := &fakeS3Downloader{}
		m := newTestReleaseManager(&fakeReleaseStorer{taskRuns: []*storage.ReleaseTaskRunsResult{
			s3Run("app", "build", 1, 0, "dist/app.tar", "s3://bucket/a/b/app.tar"),
			s3Run("app", "pkg", 2, 0, "pkg.zip", "s3://bucket2/pkg.zip"),
		}}, s3)

		require.NoError(t, m.DownloadOutputs(ctx, DownloadOutputsParams{ReleaseName: "r1", DestDir: dest}))

		assert.Equal(t, []download{
			{"bucket", "a/b/app.tar", filepath.Join(dest, "app.build", "app.tar")},
			{"bucket2", "pkg.zip", filepath.Join(dest, "app.pkg", "pkg.zip")},
		}, s3.downloads)

		assert.DirExists(t, filepath.Join(dest, "app.build"))
	})

	t.Run("output uploaded to multiple destinations is downloaded once", func(t *testing.T) {
		s3 := &fakeS3Downloader{}
		m := newTestReleaseManager(&fakeReleaseStorer{taskRuns: []*storage.ReleaseTaskRunsResult{
			s3Run("app", "build", 1, 7, "app.tar", "s3://first/app.tar"),
			s3Run("app", "build", 1, 7, "app.tar", "s3://second/app.tar"),
		}}, s3)

		require.NoError(t, m.DownloadOutputs(ctx, DownloadOutputsParams{DestDir: t.TempDir()}))

		require.Len(t, s3.downloads, 1)
		assert.Equal(t, "first", s3.downloads[0].bucket)
	})

	t.Run("different outputs of the same task are all downloaded", func(t *testing.T) {
		s3 := &fakeS3Downloader{}
		m := newTestReleaseManager(&fakeReleaseStorer{taskRuns: []*storage.ReleaseTaskRunsResult{
			s3Run("app", "build", 1, 1, "a.tar", "s3://b/a.tar"),
			s3Run("app", "build", 1, 2, "b.tar", "s3://b/b.tar"),
		}}, s3)

		require.NoError(t, m.DownloadOutputs(ctx, DownloadOutputsParams{DestDir: t.TempDir()}))

		assert.Len(t, s3.downloads, 2)
	})

	t.Run("task runs without outputs and non-S3 uploads are skipped", func(t *testing.T) {
		s3 := &fakeS3Downloader{}
		m := newTestReleaseManager(&fakeReleaseStorer{taskRuns: []*storage.ReleaseTaskRunsResult{
			{AppName: "app", TaskName: "noout", RunID: 1},
			{
				AppName: "app", TaskName: "img", RunID: 2, OutputID: 1, OutputName: "img",
				URI: "registry/img:1", UploadMethod: storage.UploadMethodDockerRegistry,
			},
			{
				AppName: "app", TaskName: "copy", RunID: 3, OutputID: 1, OutputName: "f",
				URI: "/tmp/f", UploadMethod: storage.UploadMethodFileCopy,
			},
		}}, s3)

		require.NoError(t, m.DownloadOutputs(ctx, DownloadOutputsParams{DestDir: t.TempDir()}))

		assert.Empty(t, s3.downloads)
	})

	t.Run("TaskIDs restricts the downloads", func(t *testing.T) {
		s3 := &fakeS3Downloader{}
		m := newTestReleaseManager(&fakeReleaseStorer{taskRuns: []*storage.ReleaseTaskRunsResult{
			s3Run("app", "build", 1, 0, "a", "s3://b/a"),
			s3Run("app", "test", 2, 0, "t", "s3://b/t"),
		}}, s3)

		require.NoError(t, m.DownloadOutputs(ctx, DownloadOutputsParams{
			DestDir: t.TempDir(),
			TaskIDs: []string{"app.test"},
		}))

		require.Len(t, s3.downloads, 1)
		assert.Equal(t, "t", s3.downloads[0].key)
	})

	t.Run("TaskID that is not part of the release is an error and nothing is downloaded", func(t *testing.T) {
		s3 := &fakeS3Downloader{}
		m := newTestReleaseManager(&fakeReleaseStorer{taskRuns: []*storage.ReleaseTaskRunsResult{
			s3Run("app", "build", 1, 0, "a", "s3://b/a"),
		}}, s3)

		err := m.DownloadOutputs(ctx, DownloadOutputsParams{
			DestDir: t.TempDir(),
			TaskIDs: []string{"app.build", "app.nope"},
		})

		require.ErrorContains(t, err, "app.nope")
		assert.Empty(t, s3.downloads)
	})

	t.Run("DownloadStartFn is called with task id, uri and destination", func(t *testing.T) {
		dest := t.TempDir()
		m := newTestReleaseManager(&fakeReleaseStorer{taskRuns: []*storage.ReleaseTaskRunsResult{
			s3Run("app", "build", 1, 0, "dir/a.tar", "s3://b/k/a.tar"),
		}}, &fakeS3Downloader{})

		var gotTID, gotURL, gotDest string
		require.NoError(t, m.DownloadOutputs(ctx, DownloadOutputsParams{
			DestDir: dest,
			DownloadStartFn: func(taskID, url, destFilepath string) {
				gotTID, gotURL, gotDest = taskID, url, destFilepath
			},
		}))

		assert.Equal(t, "app.build", gotTID)
		assert.Equal(t, "s3://b/k/a.tar", gotURL)
		assert.Equal(t, filepath.Join(dest, "app.build", "a.tar"), gotDest)
	})

	t.Run("malformed stored URI is an error", func(t *testing.T) {
		m := newTestReleaseManager(&fakeReleaseStorer{taskRuns: []*storage.ReleaseTaskRunsResult{
			s3Run("app", "build", 1, 0, "a", "s3:///nobucket"),
		}}, &fakeS3Downloader{})

		err := m.DownloadOutputs(ctx, DownloadOutputsParams{DestDir: t.TempDir()})

		require.ErrorContains(t, err, "malformed")
	})

	t.Run("download failure is returned", func(t *testing.T) {
		dlErr := errors.New("boom")
		m := newTestReleaseManager(&fakeReleaseStorer{taskRuns: []*storage.ReleaseTaskRunsResult{
			s3Run("app", "build", 1, 0, "a", "s3://b/a"),
		}}, &fakeS3Downloader{err: dlErr})

		err := m.DownloadOutputs(ctx, DownloadOutputsParams{DestDir: t.TempDir()})

		require.ErrorIs(t, err, dlErr)
	})

	t.Run("storage error is returned", func(t *testing.T) {
		stErr := errors.New("db down")
		m := newTestReleaseManager(&fakeReleaseStorer{taskRunsErr: stErr}, &fakeS3Downloader{})

		err := m.DownloadOutputs(ctx, DownloadOutputsParams{DestDir: t.TempDir()})

		require.ErrorIs(t, err, stErr)
	})

	t.Run("destination dir that cannot be created is an error", func(t *testing.T) {
		blocker := filepath.Join(t.TempDir(), "file")
		require.NoError(t, os.WriteFile(blocker, nil, 0o600))

		m := newTestReleaseManager(&fakeReleaseStorer{taskRuns: []*storage.ReleaseTaskRunsResult{
			s3Run("app", "build", 1, 0, "a", "s3://b/a"),
		}}, &fakeS3Downloader{})

		err := m.DownloadOutputs(ctx, DownloadOutputsParams{DestDir: blocker})

		require.Error(t, err)
	})
}

func TestValidateWantedTaskIDsHaveOutputs(t *testing.T) {
	runs := []*storage.ReleaseTaskRunsResult{
		s3Run("app", "build", 1, 0, "a", "s3://b/a"),
		{
			AppName: "app", TaskName: "img", RunID: 2, OutputID: 1, OutputName: "i",
			URI: "r/i", UploadMethod: storage.UploadMethodDockerRegistry,
		},
	}

	t.Run("no wanted tasks is fine", func(t *testing.T) {
		require.NoError(t, validateWantedTaskIDsHaveOutputs(runs, set.Set[string]{}))
	})

	t.Run("task with S3 output is fine", func(t *testing.T) {
		require.NoError(t, validateWantedTaskIDsHaveOutputs(runs, set.From([]string{"app.build"})))
	})

	t.Run("task without S3 output is reported", func(t *testing.T) {
		err := validateWantedTaskIDsHaveOutputs(runs, set.From([]string{"app.img"}))

		require.ErrorContains(t, err, "do not have an S3 output: app.img")
	})

	t.Run("unknown task is reported", func(t *testing.T) {
		err := validateWantedTaskIDsHaveOutputs(runs, set.From([]string{"app.nope"}))

		require.ErrorContains(t, err, "not part of the release: app.nope")
	})

	t.Run("both problems are reported in one error", func(t *testing.T) {
		err := validateWantedTaskIDsHaveOutputs(runs, set.From([]string{"app.nope", "app.img"}))

		require.ErrorContains(t, err, "not part of the release: app.nope")
		require.ErrorContains(t, err, "do not have an S3 output: app.img")
	})
}

func TestGetRelease(t *testing.T) {
	ctx := context.Background()

	st := &fakeReleaseStorer{
		metadata: []byte("meta"),
		taskRuns: []*storage.ReleaseTaskRunsResult{
			s3Run("app", "build", 10, 1, "a", "s3://b1/a"),
			{
				AppName: "app", TaskName: "build", RunID: 10, OutputID: 1, OutputName: "a",
				URI: "/mnt/a", UploadMethod: storage.UploadMethodFileCopy,
			},
			s3Run("app", "build", 10, 2, "b", "s3://b1/b"),
			{AppName: "app", TaskName: "noout", RunID: 11},
			s3Run("other", "build", 12, 0, "c", "s3://b2/c"),
		},
	}
	m := newTestReleaseManager(st, &fakeS3Downloader{})

	rel, err := m.GetRelease(ctx, "r1")
	require.NoError(t, err)

	assert.Equal(t, "r1", rel.ReleaseName)
	assert.Equal(t, []byte("meta"), rel.Metadata)
	require.Len(t, rel.Applications, 2)

	build := rel.Applications["app"].TaskRuns["build"]
	require.NotNil(t, build)
	assert.Equal(t, 10, build.RunID)
	require.Len(t, build.Outputs, 2)
	assert.Equal(t, []*ReleaseUpload{
		{UploadMethod: "s3", URI: "s3://b1/a"},
		{UploadMethod: "filecopy", URI: "/mnt/a"},
	}, build.Outputs[1].Uploads)

	noout := rel.Applications["app"].TaskRuns["noout"]
	require.NotNil(t, noout, "a task run without outputs is still part of the release")
	assert.Empty(t, noout.Outputs)

	assert.Contains(t, rel.Applications, "other")

	t.Run("metadata error is returned", func(t *testing.T) {
		e := errors.New("meta")
		_, err := newTestReleaseManager(&fakeReleaseStorer{metadataErr: e}, nil).GetRelease(ctx, "r")
		require.ErrorIs(t, err, e)
	})

	t.Run("task runs error is returned", func(t *testing.T) {
		e := errors.New("runs")
		_, err := newTestReleaseManager(&fakeReleaseStorer{taskRunsErr: e}, nil).GetRelease(ctx, "r")
		require.ErrorIs(t, err, e)
	})
}

type recordingLogger struct{ msgs []string }

func (l *recordingLogger) Debugf(format string, v ...any) {
	l.msgs = append(l.msgs, fmt.Sprintf(format, v...))
}

// All log messages of DownloadOutputs must be well formed, fmt reports
// arguments without a verb as "%!(EXTRA ...)".
func TestDownloadOutputsLogMessagesAreWellFormed(t *testing.T) {
	log := &recordingLogger{}
	m := &ReleaseManager{
		storage: &fakeReleaseStorer{taskRuns: []*storage.ReleaseTaskRunsResult{
			{AppName: "app", TaskName: "noout", RunID: 1},
			{AppName: "app", TaskName: "img", RunID: 2, OutputName: "i", URI: "r/i", UploadMethod: storage.UploadMethodDockerRegistry},
			s3Run("app", "build", 3, 1, "a", "s3://first/a"),
			s3Run("app", "build", 3, 1, "a", "s3://second/a"),
		}},
		s3clt:  &fakeS3Downloader{},
		logger: log,
	}

	require.NoError(t, m.DownloadOutputs(context.Background(), DownloadOutputsParams{DestDir: t.TempDir()}))

	require.Len(t, log.msgs, 3, "one message for each skipped record")
	for _, msg := range log.msgs {
		assert.NotContains(t, msg, "%!", msg)
	}
}
