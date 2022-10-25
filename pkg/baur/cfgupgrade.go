package baur

import (
	"fmt"
	"path/filepath"
	"strings"

	baur_old "github.com/simplesurance/baur"
	cfg_old "github.com/simplesurance/baur/cfg"

	"github.com/simplesurance/baur/v3/internal/fs"
	"github.com/simplesurance/baur/v3/internal/log"
	v4 "github.com/simplesurance/baur/v3/pkg/cfg/upgrade/v4"
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

func (u *CfgUpgrader) upgradeAppConfigs(
	repoRootDir string,
	includesToUpgrade *map[string]struct{},
	apps []*baur_old.App,
) error {
	for _, app := range apps {
		cfgPath := filepath.Join(app.Path, baur_old.AppCfgFile)
		appCfg, err := cfg_old.AppFromFile(cfgPath)
		if err != nil {
			return fmt.Errorf("reading application config %q failed: %w", cfgPath, err)
		}

		if err := appCfg.Validate(); err != nil {
			if appCfg.Name != "" {
				return fmt.Errorf("%s: %s", appCfg.Name, err)
			}

			return fmt.Errorf("%s: %s", cfgPath, err)
		}

		newAppCfg := v4.UpgradeAppConfig(appCfg)

		if err := fs.BackupFile(cfgPath); err != nil {
			return err
		}

		if err := newAppCfg.ToFile(cfgPath); err != nil {
			return err
		}

		log.Debugf("%s: updated", cfgPath)

		for _, includePath := range appCfg.Build.Includes {
			path := strings.ReplaceAll(includePath, "$ROOT", repoRootDir)
			if !filepath.IsAbs(path) {
				path = filepath.Join(app.Path, path)
			}

			(*includesToUpgrade)[path] = struct{}{}
		}
	}

	return nil
}

// Upgrade upgrades all baur configuration files.
// Of all changed files a backup copy is created with the same filename and a
// ".bak" suffix.
// Each upgraded configuration file is validated by running the responsible
// validate() method from the cfg package.
func (u *CfgUpgrader) Upgrade() error {
	const oldUpgradeVer = 4

	includesToUpgrade := map[string]struct{}{}
	repoCfgPath := filepath.Join(u.repoRootDir, baur_old.RepositoryCfgFile)

	oldRepoCfg, err := cfg_old.RepositoryFromFile(repoCfgPath)
	if err != nil {
		return fmt.Errorf("loading repository config %q failed: %w", repoCfgPath, err)
	}

	if oldRepoCfg.ConfigVersion != oldUpgradeVer {
		return fmt.Errorf("repository config (%q) has version %d, only version %d can be upgraded",
			repoCfgPath, oldRepoCfg.ConfigVersion, oldUpgradeVer)
	}

	oldRepo, err := baur_old.NewRepository(repoCfgPath)
	if err != nil {
		return err
	}

	// Apps are loaded to ensure their configuration and their includes
	// are valid.
	apps, err := oldRepo.FindApps()
	if err != nil {
		return err
	}

	if err := u.upgradeAppConfigs(oldRepo.Path, &includesToUpgrade, apps); err != nil {
		return err
	}

	for includePath := range includesToUpgrade {
		oldInclude, err := cfg_old.IncludeFromFile(includePath)
		if err != nil {
			return err
		}

		if err := oldInclude.Validate(); err != nil {
			return fmt.Errorf("%s: %s", includePath, err)
		}

		newInclude := v4.UpgradeIncludeConfig(oldInclude)

		if err := fs.BackupFile(includePath); err != nil {
			return err
		}

		if err := newInclude.ToFile(includePath); err != nil {
			return err
		}

		log.Debugf("%s: updated", includePath)
	}

	newRepoCfg := v4.UpgradeRepositoryConfig(oldRepoCfg)

	if err := fs.BackupFile(repoCfgPath); err != nil {
		return err
	}

	if err := newRepoCfg.ToFile(repoCfgPath); err != nil {
		return fmt.Errorf("writing new repository config file to disk failed: %w", err)
	}

	return nil
}
