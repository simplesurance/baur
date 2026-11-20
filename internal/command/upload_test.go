package command

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v5/pkg/baur"
	"github.com/simplesurance/baur/v5/pkg/cfg"
)

type fakeDockerUploader struct {
	uploads int
}

func (u *fakeDockerUploader) Upload(_, registryAddr, repository, tag string) (string, error) {
	u.uploads++
	return registryAddr + "/" + repository + ":" + tag, nil
}

type failingSizeDockerClient struct{}

func (failingSizeDockerClient) SizeBytes(string) (int64, error) {
	return 0, errors.New("inspect failed")
}

func (failingSizeDockerClient) Exists(string) (bool, error) {
	return true, nil
}

// When the size of an uploaded output can not be determined, the task run
// must not be recorded, otherwise it is stored without that output and
// later baur runs skip the task.
func TestUploadAndRecordSizeBytesErrorDoesNotStoreRun(t *testing.T) {
	redirectOutputToLogger(t)

	iidFile := filepath.Join(t.TempDir(), "iid")
	require.NoError(t, os.WriteFile(iidFile, []byte("sha256:"+strings.Repeat("a", 64)), 0o600))

	output, err := baur.NewOutputDockerImageFromIIDFile(
		failingSizeDockerClient{},
		"image",
		iidFile,
		[]*baur.UploadInfoDocker{{DockerImageRegistryUpload: &cfg.DockerImageRegistryUpload{
			Registry:   "registry.local:5000",
			Repository: "team/app",
			Tag:        "1.0",
		}}},
	)
	require.NoError(t, err)

	dockerUploader := &fakeDockerUploader{}
	// storage and gitRepo are nil: reaching baur.StoreRun() panics.
	c := &runCmd{uploader: baur.NewUploader(dockerUploader, nil, nil)}
	pt := &pendingTask{task: &baur.Task{ID: "app.build"}}

	require.NotPanics(t, func() {
		err = c.uploadAndRecord(context.Background(), pt, []baur.Output{output}, &baur.RunResult{})
	}, "task run was recorded")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "inspect failed")
	assert.Equal(t, 1, dockerUploader.uploads)
}
