package command

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v1/internal/command/flag"
	"github.com/simplesurance/baur/v1/internal/testutils/gittest"
)

func TestUpgrade(t *testing.T) {
	const (
		gitURL = "https://github.com/simplesurance/baur-example.git"
		commit = "b44698f4f90dfdc479644b888ae573fda7796e85"
	)

	initTest(t)

	tempdir := t.TempDir()

	gitDir := filepath.Join(tempdir, "git")

	require.NoError(t, os.Chdir("/"))
	gittest.Clone(t, gitDir, gitURL, commit)

	stdoutBuf, stderrBuf := interceptCmdOutput()

	require.NoError(t, os.Chdir(gitDir))

	upgradeCmd := newUpgradeConfigsCmd()
	upgradeCmd.Command.Run(&upgradeCmd.Command, nil)

	output := stdoutBuf.String()
	t.Log(output)

	require.NotNil(t, output)
	require.Contains(t, output, "successful", "command did not log a success message, output was: %q", output)

	stderrOut := stderrBuf.String()
	require.Empty(t, stderrOut, "command wrote something to stderr: %q", stderrOut)

	stdoutBuf.Truncate(0)
	statusCmd := newStatusCmd()
	statusCmd.csv = true
	statusCmd.fields = &flag.Fields{Fields: []string{statusNameParam}}
	statusCmd.Command.Run(&statusCmd.Command, nil)

	taskIDs := strings.Split(strings.TrimSpace(stdoutBuf.String()), "\n")

	assert.Len(t, taskIDs, 4, taskIDs)
	assert.Contains(t, taskIDs, "hello-server.build")
	assert.Contains(t, taskIDs, "myredis.build")
	assert.Contains(t, taskIDs, "random.build")
	assert.Contains(t, taskIDs, "unixtime.build")
}
