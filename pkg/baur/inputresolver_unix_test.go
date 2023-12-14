//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package baur

import (
	"os"
	"os/user"
	"strconv"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v3/internal/log"
)

func TestSymlinkTargetFileOwnerChange(t *testing.T) {
	t.Skip("fails because of bug: https://github.com/simplesurance/baur/issues/493")
	log.RedirectToTestingLog(t)

	user, err := user.Current()
	require.NoError(t, err)
	gids, err := user.GroupIds()
	require.NoError(t, err)
	require.NotEmpty(t, gids, "requiring the user to be at least in 1 additional group, to run the testcase")

	info := prepareSymlinkTestDir(t, false, false)
	require.NoError(t, err)

	fi, err := os.Stat(info.SymlinkTargetFilePath)
	require.NoError(t, err)
	statt := fi.Sys().(*syscall.Stat_t)
	currentOwnerGid := statt.Gid

	newGid := -1
	for _, g := range gids {
		gidNr, err := strconv.Atoi(g)
		require.NoError(t, err)

		if int64(gidNr) != int64(currentOwnerGid) {
			newGid = gidNr
			break
		}
	}
	require.NotEqualf(t, -1, newGid, "could not find a supplementary group of the user that is the current group owner of the file (%v) ", currentOwnerGid)

	require.NoError(t, os.Chown(info.SymlinkTargetFilePath, -1, newGid))
	t.Logf("changed group owner of %v to %v", info.SymlinkTargetFilePath, newGid)

	digestAfter := resolveInputs(t, info.Task)
	require.NotEqual(t, info.TotalInputDigest.String(), digestAfter.String())
}
