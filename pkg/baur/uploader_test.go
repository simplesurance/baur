package baur

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v5/pkg/cfg"
)

type uploadCall struct{ kind, a, b, c, d string }

type fakeUploaders struct {
	calls []uploadCall
	err   map[string]error // by kind
}

func (f *fakeUploaders) record(kind, a, b, c, d string) error {
	f.calls = append(f.calls, uploadCall{kind, a, b, c, d})
	return f.err[kind]
}

type fakeDocker struct{ *fakeUploaders }

func (f fakeDocker) Upload(image, registry, repo, tag string) (string, error) {
	return "docker://" + repo + ":" + tag, f.record("docker", image, registry, repo, tag)
}

type fakeS3 struct{ *fakeUploaders }

func (f fakeS3) Upload(_ context.Context, path, bucket, key string) (string, error) {
	return "s3://" + bucket + "/" + key, f.record("s3", path, bucket, key, "")
}

type fakeFileCopy struct{ *fakeUploaders }

func (f fakeFileCopy) Upload(src, dst string) (string, error) {
	return "file://" + dst, f.record("filecopy", src, dst, "", "")
}

func newTestUploader() (*Uploader, *fakeUploaders) {
	f := &fakeUploaders{err: map[string]error{}}
	return NewUploader(fakeDocker{f}, fakeS3{f}, fakeFileCopy{f}), f
}

func TestUploaderUploadsFileToAllDestinations(t *testing.T) {
	u, f := newTestUploader()

	out := NewOutputFile("dist/app.tar", "/repo/app/dist/app.tar",
		[]*UploadInfoS3{
			{S3Upload: &cfg.S3Upload{Bucket: "b1", Key: "k1"}},
			{S3Upload: &cfg.S3Upload{Bucket: "b2", Key: "k2"}},
		},
		[]*UploadInfoFileCopy{{FileCopy: &cfg.FileCopy{Path: "/mnt/out"}}},
	)

	var started, results int
	var gotResults []*UploadResult

	err := u.Upload(context.Background(), out,
		func(Output, UploadInfo) { started++ },
		func(o Output, r *UploadResult) {
			results++
			assert.Same(t, out, o)
			gotResults = append(gotResults, r)
		},
	)
	require.NoError(t, err)

	assert.Equal(t, 3, started)
	assert.Equal(t, 3, results)

	// filecopy destinations are processed before S3 destinations
	assert.Equal(t, []uploadCall{
		{"filecopy", "/repo/app/dist/app.tar", filepath.Join("/mnt/out", "app.tar"), "", ""},
		{"s3", "/repo/app/dist/app.tar", "b1", "k1", ""},
		{"s3", "/repo/app/dist/app.tar", "b2", "k2", ""},
	}, f.calls)

	require.Len(t, gotResults, 3)
	assert.Equal(t, UploadMethodFilecopy, gotResults[0].Method)
	assert.Equal(t, "file://"+filepath.Join("/mnt/out", "app.tar"), gotResults[0].URL)
	assert.Equal(t, UploadMethodS3, gotResults[1].Method)
	assert.Equal(t, "s3://b1/k1", gotResults[1].URL)
	assert.False(t, gotResults[1].Stop.Before(gotResults[1].Start))
}

func TestUploaderUploadsDockerImage(t *testing.T) {
	u, f := newTestUploader()

	out := &OutputDockerImage{
		ImageID: "sha256:abc",
		UploadDestinations: []*UploadInfoDocker{
			{DockerImageRegistryUpload: &cfg.DockerImageRegistryUpload{Registry: "reg:5000", Repository: "r/i", Tag: "v1"}},
		},
	}

	var res *UploadResult
	err := u.Upload(context.Background(), out, func(Output, UploadInfo) {}, func(_ Output, r *UploadResult) { res = r })
	require.NoError(t, err)

	assert.Equal(t, []uploadCall{{"docker", "sha256:abc", "reg:5000", "r/i", "v1"}}, f.calls)
	require.NotNil(t, res)
	assert.Equal(t, UploadMethodDocker, res.Method)
	assert.Equal(t, "docker://r/i:v1", res.URL)
}

func TestUploaderErrors(t *testing.T) {
	nop := func(Output, UploadInfo) {}
	nopRes := func(Output, *UploadResult) {}

	t.Run("docker image without upload destinations (nil) is an error", func(t *testing.T) {
		u, _ := newTestUploader()
		err := u.Upload(context.Background(), &OutputDockerImage{}, nop, nopRes)
		require.ErrorContains(t, err, "uploadDestination is nil")
	})

	t.Run("unsupported output type", func(t *testing.T) {
		u, _ := newTestUploader()
		err := u.Upload(context.Background(), unsupportedOutput{}, nop, nopRes)
		require.ErrorContains(t, err, "unsupported output type")
	})

	for _, kind := range []string{"docker", "filecopy", "s3"} {
		t.Run(kind+" failure is wrapped and stops further uploads", func(t *testing.T) {
			u, f := newTestUploader()
			boom := errors.New("boom")
			f.err[kind] = boom

			var out Output
			switch kind {
			case "docker":
				out = &OutputDockerImage{UploadDestinations: []*UploadInfoDocker{
					{DockerImageRegistryUpload: &cfg.DockerImageRegistryUpload{Repository: "r", Tag: "1"}},
					{DockerImageRegistryUpload: &cfg.DockerImageRegistryUpload{Repository: "r", Tag: "2"}},
				}}
			case "filecopy":
				out = NewOutputFile("f", "/a/f", nil, []*UploadInfoFileCopy{
					{FileCopy: &cfg.FileCopy{Path: "/x"}}, {FileCopy: &cfg.FileCopy{Path: "/y"}},
				})
			case "s3":
				out = NewOutputFile("f", "/a/f", []*UploadInfoS3{
					{S3Upload: &cfg.S3Upload{Bucket: "b", Key: "1"}}, {S3Upload: &cfg.S3Upload{Bucket: "b", Key: "2"}},
				}, nil)
			}

			var results int
			err := u.Upload(context.Background(), out, nop, func(Output, *UploadResult) { results++ })

			require.ErrorIs(t, err, boom)
			assert.Contains(t, err.Error(), kind)
			assert.Zero(t, results, "no result callback for the failed upload")
			assert.Len(t, f.calls, 1, "uploads after the first failure must not run")
		})
	}
}

type unsupportedOutput struct{ Output }
