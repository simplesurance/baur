//go:build dbtest

package command

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	oldpostgres "github.com/simplesurance/baur/v2/pkg/storage/postgres"

	"github.com/simplesurance/baur/v3/internal/testutils/repotest"
)

func TestUpgradeDb_DatabaseNotExist(t *testing.T) {
	var exitCode int

	initTest(t)
	_ = repotest.CreateBaurRepository(t, repotest.WithNewDB())
	_, stderrBuf := interceptCmdOutput(t)

	oldExitFunc := exitFunc
	exitFunc = func(code int) {
		exitCode = code
	}
	t.Cleanup(func() {
		exitFunc = oldExitFunc
	})

	upgradeDbCmd := newUpgradeDatabaseCmd()
	upgradeDbCmd.Command.Run(&upgradeDbCmd.Command, nil)

	assert.Equal(t, 1, exitCode)
	assert.Contains(t, stderrBuf.String(), "database not found")
}

func TestUpgradeDb_AlreadyUptodate(t *testing.T) {
	initTest(t)
	_ = repotest.CreateBaurRepository(t, repotest.WithNewDB())
	stdoutBuf, _ := interceptCmdOutput(t)

	initDbCmd.Run(initDbCmd, nil)

	upgradeDbCmd := newUpgradeDatabaseCmd()
	upgradeDbCmd.Command.Run(&upgradeDbCmd.Command, nil)

	assert.Contains(t, stdoutBuf.String(), "already up to date")
}

func TestUpgradeDb(t *testing.T) {
	initTest(t)
	_ = repotest.CreateBaurRepository(t, repotest.WithNewDB())

	repo := mustFindRepository()
	uri := mustGetPSQLURI(repo.Cfg)

	oldDbClt, err := oldpostgres.New(ctx, uri, nil)
	require.NoError(t, err)
	err = oldDbClt.Init(ctx)
	require.NoError(t, err)

	stdoutBuf, _ := interceptCmdOutput(t)
	upgradeDbCmd := newUpgradeDatabaseCmd()
	upgradeDbCmd.Command.Run(&upgradeDbCmd.Command, nil)

	assert.Contains(t, stdoutBuf.String(), "database schema successfully upgraded from version 1 to 2")
}
