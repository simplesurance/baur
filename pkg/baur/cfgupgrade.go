package baur

import (
	"fmt"
	"path/filepath"

	"github.com/simplesurance/baur/v5/internal/fs"
	"github.com/simplesurance/baur/v5/internal/log"
	"github.com/simplesurance/baur/v5/pkg/cfg"
	v5 "github.com/simplesurance/baur/v5/pkg/cfg/upgrade/v5"
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

	if repoCfg.ConfigVersion != 5 && repoCfg.ConfigVersion != 6 {
		return fmt.Errorf("%q specifies 'config_version=%d', upgrading is only supported from version 5 and 6", repoCfgPath, repoCfg.ConfigVersion)
	}

	// the repository config of version 5, 6 and 7 are the same 7 only the
	// ConfigVersion value changes
	if err := u.upgradeCfgVersion(repoCfgPath); err != nil {
		return fmt.Errorf("upgrading configuration files from version %d to %d failed: %w", repoCfg.ConfigVersion, cfg.Version, err)
	}

	return nil
}

// IsUpToDate returns true if the configuration are up to date.
func (u *CfgUpgrader) IsUpToDate() (bool, error) {
	p := filepath.Join(u.repoRootDir, RepositoryCfgFile)
	repoCfg, err := cfg.RepositoryFromFile(p)
	if err != nil {
		return false, fmt.Errorf("loading repository config %q failed: %w", p, err)
	}

	return repoCfg.ConfigVersion == cfg.Version, nil
}

func (u *CfgUpgrader) upgradeCfgVersion(repoCfgPath string) error {
	oldRepoCfg, err := cfg.RepositoryFromFile(repoCfgPath)
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

	log.Debugf("%s: was updated from v%d to v%d", repoCfgPath, oldRepoCfg.ConfigVersion, newRepoCfg.ConfigVersion)

	return nil
}
