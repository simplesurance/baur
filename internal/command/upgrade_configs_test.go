package command

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v5/internal/testutils/fstest"
	"github.com/simplesurance/baur/v5/internal/testutils/gittest"
	"github.com/simplesurance/baur/v5/internal/testutils/repotest"
	"github.com/simplesurance/baur/v5/pkg/baur"
	"github.com/simplesurance/baur/v5/pkg/cfg"
)

func TestUpgrade(t *testing.T) {
	initTest(t)

	repoCfgVersions := []int{5, 6}
	for _, fromVer := range repoCfgVersions {
		t.Run(fmt.Sprintf("from: %d", fromVer), func(t *testing.T) {
			initTest(t)
			repoDir := fstest.TempDir(t)
			gittest.CreateRepository(t, repoDir)
			t.Chdir(repoDir)
			fstest.MkdirAll(t, filepath.Join(repoDir, "a"))

			repoCfg := cfg.Repository{
				ConfigVersion: fromVer,
				Database:      cfg.Database{PGSQLURL: "postgres://test"},
				Discover:      cfg.Discover{Dirs: []string{"a", "simpleApp"}, SearchDepth: 3},
			}
			repoCfgPath := filepath.Join(repoDir, baur.RepositoryCfgFile)
			err := repoCfg.ToFile(repoCfgPath)
			require.NoError(t, err)

			testRepo := &repotest.Repo{Cfg: &repoCfg, Dir: repoDir}
			testRepo.CreateSimpleApp(t)

			lsAppsCmd := newLsAppsCmd()
			execCheck(t, lsAppsCmd, 1) // must fail, wrong cfg version

			stdoutBuf, stderrBuf := interceptCmdOutput(t)

			upgradeCmd := newUpgradeConfigsCmd()
			upgradeCmd.Run(&upgradeCmd.Command, nil)

			output := stdoutBuf.String()
			t.Log(output)
			require.NotNil(t, output)
			require.Contains(t, output, "successful", "command did not log a success message, output was: %q", output)

			stderrOut := stderrBuf.String()
			require.Empty(t, stderrOut, "command wrote something to stderr: %q", stderrOut)

			execCheck(t, lsAppsCmd, 0) // must fail, wrong cfg version

			repoCfgLoaded, err := cfg.RepositoryFromFile(repoCfgPath)
			require.NoError(t, err)
			require.EqualValues(t, repoCfg.Database, repoCfgLoaded.Database)
			require.EqualValues(t, repoCfg.Discover, repoCfgLoaded.Discover)
		})
	}
}
