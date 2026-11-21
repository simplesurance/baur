//go:build dbtest

package command

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v5/internal/testutils/fstest"
)

func createRelease(t *testing.T, expectedExitCode int, args ...string) {
	t.Helper()

	cmd := newReleaseCreateCmd()
	cmd.SetArgs(args)
	execCheck(t, cmd, expectedExitCode)
}

func TestReleaseCreateIncludeAndMetadata(t *testing.T) {
	r := newRepoWithTasks(t, shTask("a", "true", nil), shTask("b", "true", nil))
	runTasks(t, 0)

	metadataFile := filepath.Join(t.TempDir(), "meta.txt")
	require.NoError(t, os.WriteFile(metadataFile, []byte("build-42"), 0o600))

	createRelease(t, 0, "--include", "svc.a", "--metadata", metadataFile, "partial")

	assert.True(t, releaseExists(t, r, "partial"))

	stdoutBuf, _ := interceptCmdOutput(t)
	showCmd := newReleaseShowCmd()
	showCmd.SetArgs([]string{"partial"})
	execCheck(t, showCmd, 0)

	release := struct {
		ReleaseName  string
		Metadata     []byte
		Applications map[string]struct{ TaskRuns map[string]any }
	}{}
	require.NoError(t, json.Unmarshal(stdoutBuf.Bytes(), &release), stdoutBuf.String())

	assert.Equal(t, "partial", release.ReleaseName)
	assert.Equal(t, "build-42", string(release.Metadata))
	require.Contains(t, release.Applications, "svc")
	assert.Len(t, release.Applications["svc"].TaskRuns, 1, "only included tasks are part of the release")
	assert.Contains(t, release.Applications["svc"].TaskRuns, "a")

	t.Run("metadata can be exported", func(t *testing.T) {
		out := filepath.Join(t.TempDir(), "exported.txt")
		showCmd := newReleaseShowCmd()
		showCmd.SetArgs([]string{"--metadata", out, "partial"})
		execCheck(t, showCmd, 0)

		content, err := os.ReadFile(out)
		require.NoError(t, err)
		assert.Equal(t, "build-42", string(content))
	})
}

func TestReleaseCreateErrors(t *testing.T) {
	r := newRepoWithTasks(t, shTask("a", "true", nil))
	runTasks(t, 0)
	createRelease(t, 0, "existing")

	t.Run("name with leading whitespace is rejected", func(t *testing.T) {
		_, stderrBuf := interceptCmdOutput(t)

		createRelease(t, exitCodeError, " bad")

		assert.Contains(t, stderrBuf.String(), "invalid release name")
		assert.False(t, releaseExists(t, r, " bad"))
	})

	t.Run("duplicate name", func(t *testing.T) {
		_, stderrBuf := interceptCmdOutput(t)

		createRelease(t, exitCodeAlreadyExist, "existing")

		assert.Contains(t, stderrBuf.String(), "already exists")
	})

	t.Run("include pattern without matches", func(t *testing.T) {
		_, stderrBuf := interceptCmdOutput(t)

		createRelease(t, exitCodeError, "--include", "nope.build", "norelease")

		assert.Contains(t, stderrBuf.String(), "nope")
		assert.False(t, releaseExists(t, r, "norelease"))
	})

	t.Run("metadata file that does not exist", func(t *testing.T) {
		_, stderrBuf := interceptCmdOutput(t)

		createRelease(t, exitCodeError, "--metadata", filepath.Join(t.TempDir(), "missing"), "nometa")

		assert.Contains(t, stderrBuf.String(), "opening metadata file failed")
		assert.False(t, releaseExists(t, r, "nometa"))
	})

	t.Run("pending task", func(t *testing.T) {
		require.NoError(t, os.WriteFile(filepath.Join(r.Dir, "svc", ".app.toml"),
			append(fstest.ReadFile(t, filepath.Join(r.Dir, "svc", ".app.toml")), []byte("\n# changed\n")...), 0o600))

		createRelease(t, exitCodeTaskRunIsPending, "pendingrelease")

		assert.False(t, releaseExists(t, r, "pendingrelease"))
	})
}

func TestReleaseCreateWithoutTasks(t *testing.T) {
	r := newRepoWithTasks(t)
	_, stderrBuf := interceptCmdOutput(t)

	createRelease(t, exitCodeError, "empty")

	assert.Contains(t, stderrBuf.String(), "could not find any tasks")
	assert.False(t, releaseExists(t, r, "empty"))
}
