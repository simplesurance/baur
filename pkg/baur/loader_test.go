package baur

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v3/internal/log"
)

func TestFindAppConfigsRemovesDups(t *testing.T) {
	log.RedirectToTestingLog(t)

	repoDir := filepath.Join(testdataDir, "app_matches_multiple_app_dirs")
	err := os.Chdir(repoDir)
	require.NoError(t, err)

	searchDirs := []string{".", "app1", "app1/.", "app1/..", "app1/../app1/."}

	result, err := findAppConfigs(searchDirs, 5, log.StdLogger)
	require.NoError(t, err)

	require.Len(t, result, 1)
}
