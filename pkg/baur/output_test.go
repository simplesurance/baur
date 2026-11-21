package baur

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v5/pkg/cfg"
)

const testImageID = "sha256:2cbc2a1d1b2c2b5f3a3b3b8f0c5a0e2f9a1b7c9d8e6f5a4b3c2d1e0f9a8b7c6d"

func TestOutputTypeString(t *testing.T) {
	assert.Equal(t, "docker", DockerOutput.String())
	assert.Equal(t, "file", FileOutput.String())
	assert.Equal(t, "invalid OutputType", OutputType(99).String())
}

func TestOutputsFromTask(t *testing.T) {
	dockerClient := &fakeDockerInfoClient{}

	t.Run("task without outputs returns no outputs", func(t *testing.T) {
		got, err := OutputsFromTask(dockerClient, &Task{Outputs: &cfg.Output{}})
		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("docker outputs are returned before file outputs", func(t *testing.T) {
		dir := t.TempDir()
		// docker writes the iidfile without a trailing newline
		require.NoError(t, os.WriteFile(filepath.Join(dir, "image.id"), []byte(testImageID), 0o600))

		task := &Task{
			Directory: dir,
			Outputs: &cfg.Output{
				File: []cfg.FileOutput{{
					Path:     "dist/app.tar",
					S3Upload: []cfg.S3Upload{{Bucket: "b", Key: "k"}},
					FileCopy: []cfg.FileCopy{{Path: "/tmp/out"}},
				}},
				DockerImage: []cfg.DockerImageOutput{{
					IDFile: "image.id",
					RegistryUpload: []cfg.DockerImageRegistryUpload{
						{Repository: "repo/img", Tag: "v1"},
						{Registry: "reg:5000", Repository: "repo/img", Tag: "v2"},
					},
				}},
			},
		}

		got, err := OutputsFromTask(dockerClient, task)
		require.NoError(t, err)
		require.Len(t, got, 2)

		assert.Equal(t, DockerOutput, got[0].Type())
		assert.Equal(t, "image.id", got[0].Name())
		img, ok := got[0].(*OutputDockerImage)
		require.True(t, ok)
		assert.Equal(t, testImageID, img.ImageID)
		require.Len(t, img.UploadDestinations, 2)
		assert.Equal(t, "v2", img.UploadDestinations[1].Tag)

		assert.Equal(t, FileOutput, got[1].Type())
		assert.Equal(t, "dist/app.tar", got[1].Name())
		f, ok := got[1].(*OutputFile)
		require.True(t, ok)
		assert.Equal(t, filepath.Join(dir, "dist/app.tar"), f.AbsPath())
	})

	t.Run("file output without upload method is an error", func(t *testing.T) {
		task := &Task{Outputs: &cfg.Output{File: []cfg.FileOutput{{Path: "dist/app.tar"}}}}

		_, err := OutputsFromTask(dockerClient, task)

		require.ErrorContains(t, err, "no upload method")
		require.ErrorContains(t, err, "dist/app.tar")
	})

	t.Run("missing image id file is an error", func(t *testing.T) {
		task := &Task{
			Directory: t.TempDir(),
			Outputs:   &cfg.Output{DockerImage: []cfg.DockerImageOutput{{IDFile: "nope.id"}}},
		}

		_, err := OutputsFromTask(dockerClient, task)

		require.ErrorContains(t, err, "nope.id")
	})

	t.Run("invalid image id is an error", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "image.id"), []byte("not-a-digest\n"), 0o600))
		task := &Task{
			Directory: dir,
			Outputs:   &cfg.Output{DockerImage: []cfg.DockerImageOutput{{IDFile: "image.id"}}},
		}

		_, err := OutputsFromTask(dockerClient, task)

		require.ErrorContains(t, err, "invalid format")
	})
}

// Some tools write a trailing newline to the iidfile, it is not part of the image ID.
func TestNewOutputDockerImageFromIIDFileTrimsTrailingNewline(t *testing.T) {
	iid := filepath.Join(t.TempDir(), "image.id")
	require.NoError(t, os.WriteFile(iid, []byte(testImageID+"\n"), 0o600))

	img, err := NewOutputDockerImageFromIIDFile(&fakeDockerInfoClient{}, "image.id", iid, nil)
	require.NoError(t, err)

	assert.Equal(t, testImageID, img.ImageID)
}
