package baur

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v3/internal/log"
	"github.com/simplesurance/baur/v3/internal/testutils/ostest"
	"github.com/simplesurance/baur/v3/pkg/cfg"
)

func TestFindAppConfigsRemovesDups(t *testing.T) {
	log.RedirectToTestingLog(t)

	repoDir := filepath.Join(testdataDir, "app_matches_multiple_app_dirs")
	ostest.Chdir(t, repoDir)

	searchDirs := []string{".", "app1", "app1/.", "app1/..", "app1/../app1/."}

	result, err := findAppConfigs(repoDir, searchDirs, 5, log.StdLogger)
	require.NoError(t, err)

	require.Len(t, result, 1)
}

func TestErrorOnDuplicateAppNames(t *testing.T) {
	log.RedirectToTestingLog(t)
	repoDir := filepath.Join(testdataDir, "duplicate_app_names")

	repoCfg, err := cfg.RepositoryFromFile(filepath.Join(repoDir, RepositoryCfgFile))
	require.NoError(t, err)

	loader, err := NewLoader(repoCfg, nil, log.StdLogger)
	require.NoError(t, err)

	wantedErr := &ErrDuplicateAppNames{}
	_, err = loader.LoadApps("*")
	require.ErrorAs(t, err, &wantedErr)

	_, err = loader.LoadTasks("*.*")
	require.ErrorAs(t, err, &wantedErr)

	_, err = loader.LoadTasks("*")
	require.ErrorAs(t, err, &wantedErr)

	// because we abort the search and do not load other configs if the
	// apps or tasks found matching the target, no error is returned in
	// these scenarios:
	// _, err = loader.LoadApps("app1")
	// require.ErrorAs(t, err, &wantedErr)

	// _, err = loader.LoadTasks("app1.*")
	// require.ErrorAs(t, err, &wantedErr)

	// _, err = loader.LoadTasks("app1.build")
	// require.ErrorAs(t, err, &wantedErr)
}
