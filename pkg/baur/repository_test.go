package baur

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v3/internal/fs"
	"github.com/simplesurance/baur/v3/internal/log"
	"github.com/simplesurance/baur/v3/pkg/cfg"
)

func TestNewRepositoryUsesRealPaths(t *testing.T) {
	log.RedirectToTestingLog(t)

	repoDir := t.TempDir()
	repoDir, err := fs.RealPath(repoDir)
	require.NoError(t, err, "calcuting realpath of temp dir failed")

	symlinkPath := filepath.Join(os.TempDir(), "baur_test-"+uuid.New().String())
	t.Cleanup(func() {
		_ = os.Remove(symlinkPath)
	})

	err = os.Symlink(repoDir, symlinkPath)
	require.NoError(t, err, "creating symlink failed %s -> %s", symlinkPath, repoDir)

	cfgR := cfg.Repository{
		ConfigVersion: cfg.Version,

		Discover: cfg.Discover{
			Dirs:        []string{"."},
			SearchDepth: 10,
		},
	}

	baurCfgPath := filepath.Join(repoDir, RepositoryCfgFile)
	if err := cfgR.ToFile(baurCfgPath); err != nil {
		t.Fatalf("could not write repository cfg file: %s", err)
	}

	r, err := NewRepository(filepath.Join(symlinkPath, RepositoryCfgFile))
	require.NoError(t, err, "NewRepository failed")

	assert.Equal(t, repoDir, r.Path)
	assert.Equal(t, baurCfgPath, r.CfgPath)
}
