package baur

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeDockerInfoClient struct {
	size    int64
	sizeErr error
}

func (c *fakeDockerInfoClient) SizeBytes(string) (int64, error) {
	return c.size, c.sizeErr
}

func (*fakeDockerInfoClient) Exists(string) (bool, error) {
	return true, nil
}

func TestOutputDockerImageSizeBytes(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		img := &OutputDockerImage{dockerClient: &fakeDockerInfoClient{size: 528}}

		size, err := img.SizeBytes()
		require.NoError(t, err)
		assert.Equal(t, uint64(528), size)
	})

	t.Run("error", func(t *testing.T) {
		img := &OutputDockerImage{dockerClient: &fakeDockerInfoClient{sizeErr: errors.New("inspect failed")}}

		_, err := img.SizeBytes()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "inspect failed")
	})
}
