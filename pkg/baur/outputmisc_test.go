package baur

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v5/pkg/cfg"
)

func TestOutputFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "app.tar")
	require.NoError(t, os.WriteFile(p, []byte("hello"), 0o600))

	out := NewOutputFile("app.tar", p, nil, nil)

	t.Run("accessors", func(t *testing.T) {
		assert.Equal(t, "app.tar", out.Name())
		assert.Equal(t, p, out.AbsPath())
		assert.Equal(t, FileOutput, out.Type())
	})

	t.Run("Digest is the sha384 of the content only", func(t *testing.T) {
		d, err := out.Digest()
		require.NoError(t, err)

		// sha384sum of "hello"
		assert.Equal(t, "sha384:59e1748777448c69de6b800d7a33bbfb9ff1b463e44354c3553bcdb9c666fa90125a3c79f90397bdf5f6a13de828684f", d.String())
	})

	t.Run("Digest is cached", func(t *testing.T) {
		d1, err := out.Digest()
		require.NoError(t, err)

		require.NoError(t, os.WriteFile(p, []byte("changed"), 0o600))
		d2, err := out.Digest()
		require.NoError(t, err)

		assert.Same(t, d1, d2)

		recalculated, err := out.CalcDigest()
		require.NoError(t, err)
		assert.NotEqual(t, d1.String(), recalculated.String(), "CalcDigest always reads the file")
	})

	t.Run("SizeBytes and Exists", func(t *testing.T) {
		require.NoError(t, os.WriteFile(p, []byte("12345"), 0o600))

		size, err := out.SizeBytes()
		require.NoError(t, err)
		assert.Equal(t, uint64(5), size)

		ok, err := out.Exists()
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("missing file", func(t *testing.T) {
		missing := NewOutputFile("x", filepath.Join(dir, "missing"), nil, nil)

		ok, err := missing.Exists()
		require.NoError(t, err)
		assert.False(t, ok)

		_, err = missing.SizeBytes()
		require.Error(t, err)

		_, err = missing.Digest()
		require.Error(t, err)
	})
}

func TestOutputDockerImageDigestAndExists(t *testing.T) {
	dir := t.TempDir()
	iid := filepath.Join(dir, "image.id")
	require.NoError(t, os.WriteFile(iid, []byte(testImageID), 0o600))

	img, err := NewOutputDockerImageFromIIDFile(&fakeDockerInfoClient{}, "image.id", iid, nil)
	require.NoError(t, err)

	d, err := img.Digest()
	require.NoError(t, err)
	assert.Equal(t, testImageID, d.String(), "the digest is the image ID")

	ok, err := img.Exists()
	require.NoError(t, err)
	assert.True(t, ok)

	assert.Equal(t, DockerOutput, img.Type())
	assert.Equal(t, "image.id", img.Name())
	assert.Equal(t, "docker image: "+testImageID, img.String())
}

func TestUploadInfoString(t *testing.T) {
	assert.Equal(t, "s3://bucket/key", (&UploadInfoS3{S3Upload: &cfg.S3Upload{Bucket: "bucket", Key: "key"}}).String())
	assert.Equal(t, "/mnt/out", (&UploadInfoFileCopy{FileCopy: &cfg.FileCopy{Path: "/mnt/out"}}).String())

	t.Run("docker without registry", func(t *testing.T) {
		u := &UploadInfoDocker{DockerImageRegistryUpload: &cfg.DockerImageRegistryUpload{Repository: "r/i", Tag: "v1"}}
		assert.Equal(t, "r/i:v1", u.String())
	})

	t.Run("docker with registry", func(t *testing.T) {
		u := &UploadInfoDocker{DockerImageRegistryUpload: &cfg.DockerImageRegistryUpload{
			Registry: "reg:5000", Repository: "r/i", Tag: "v1",
		}}
		assert.Equal(t, "reg:5000/r/i:v1", u.String())
	})
}

func TestTaskStatusString(t *testing.T) {
	assert.Equal(t, "Undefined", TaskStatusUndefined.String())
	assert.Equal(t, "Exist", TaskStatusRunExist.String())
	assert.Equal(t, "Pending", TaskStatusExecutionPending.String())
	assert.Panics(t, func() { _ = TaskStatus(99).String() })
}
