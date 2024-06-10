package baur

import (
	"fmt"
	"path/filepath"

	cfg_v5 "github.com/simplesurance/baur/v2/pkg/cfg"

	"github.com/simplesurance/baur/v3/internal/fs"
	"github.com/simplesurance/baur/v3/internal/log"
	"github.com/simplesurance/baur/v3/pkg/cfg"
	v5 "github.com/simplesurance/baur/v3/pkg/cfg/upgrade/v5"
)

// CfgUpgrader converts baur configurations files from a previous format to the
// current one.
type CfgUpgrader struct {
	newIncludeID string
	repoRootDir  string
}

// NewCfgUpgrader returns an new CfgUpgrader to upgrade configuration files in
// repositoryRootDir.
func NewCfgUpgrader(repositoryRootDir string) *CfgUpgrader {
	return &CfgUpgrader{
		repoRootDir:  repositoryRootDir,
		newIncludeID: "main",
	}
}

// Upgrade upgrades all baur configuration files.
// Of all changed files a backup copy is created with the same filename and a
// ".bak" suffix.
// Each upgraded configuration file is validated by running the responsible
// validate() method from the cfg package.
func (u *CfgUpgrader) Upgrade() error {
	repoCfgPath := filepath.Join(u.repoRootDir, RepositoryCfgFile)
	repoCfg, err := cfg.RepositoryFromFile(repoCfgPath)
	if err != nil {
		return fmt.Errorf("loading repository config %q failed: %w", repoCfgPath, err)
	}

	if repoCfg.ConfigVersion == 0 {
		return fmt.Errorf("loading repository config %q succeeded but 'config_version' parameter is missing or 0", repoCfgPath)
	}

	if repoCfg.ConfigVersion != 5 {
		return fmt.Errorf("%q specifies 'config_version=%d', upgrading is only supported from version 5", repoCfgPath, repoCfg.ConfigVersion)
	}

	if err := u.upgradeV5(repoCfgPath); err != nil {
		return fmt.Errorf("upgrading configuration files from version %d to %d failed: %w", repoCfg.ConfigVersion, 6, err)
	}

	return nil
}

func (u *CfgUpgrader) upgradeV5(repoCfgPath string) error {
	oldRepoCfg, err := cfg_v5.RepositoryFromFile(repoCfgPath)
	if err != nil {
		return fmt.Errorf("loading repository config %q failed: %w", repoCfgPath, err)
	}

	if err := fs.BackupFile(repoCfgPath); err != nil {
		return err
	}

	newRepoCfg := v5.UpgradeRepositoryConfig(oldRepoCfg)
	if err := newRepoCfg.ToFile(repoCfgPath, cfg.ToFileOptOverwrite()); err != nil {
		return fmt.Errorf("writing new repository config file to disk failed: %w", err)
	}

	log.Debugf("%s: was updated from v5 to v6", repoCfgPath)

	return nil
}
