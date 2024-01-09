package baur

import (
	"fmt"
	"path/filepath"
	"strings"

	baur_old "github.com/simplesurance/baur"
	cfg_v4 "github.com/simplesurance/baur/cfg"
	cfg_v5 "github.com/simplesurance/baur/v2/pkg/cfg"

	"github.com/simplesurance/baur/v3/internal/fs"
	"github.com/simplesurance/baur/v3/internal/log"
	"github.com/simplesurance/baur/v3/pkg/cfg"
	v4 "github.com/simplesurance/baur/v3/pkg/cfg/upgrade/v4"
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

func (u *CfgUpgrader) upgradeAppConfigs(
	repoRootDir string,
	includesToUpgrade *map[string]struct{},
	apps []*baur_old.App,
) error {
	for _, app := range apps {
		cfgPath := filepath.Join(app.Path, baur_old.AppCfgFile)
		appCfg, err := cfg_v4.AppFromFile(cfgPath)
		if err != nil {
			return fmt.Errorf("reading application config %q failed: %w", cfgPath, err)
		}

		if err := appCfg.Validate(); err != nil {
			if appCfg.Name != "" {
				return fmt.Errorf("%s: %w", appCfg.Name, err)
			}

			return fmt.Errorf("%s: %w", cfgPath, err)
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
	repoCfgPath := filepath.Join(u.repoRootDir, RepositoryCfgFile)
	repoCfg, err := cfg.RepositoryFromFile(repoCfgPath)
	if err != nil {
		return fmt.Errorf("loading repository config %q failed: %w", repoCfgPath, err)
	}

	if repoCfg.ConfigVersion == 0 {
		return fmt.Errorf("loading repository config %q succeeded but 'config_version' parameter is missing or 0", repoCfgPath)
	}

	if repoCfg.ConfigVersion != 4 && repoCfg.ConfigVersion != 5 {
		return fmt.Errorf("%q specifies 'config_version=%d', upgrading is only supported from version 4 and 5", repoCfgPath, repoCfg.ConfigVersion)
	}

	if repoCfg.ConfigVersion == 4 {
		if err := u.upgradeV4(); err != nil {
			return fmt.Errorf("upgrading configuration files from version %d to %d failed: %w", repoCfg.ConfigVersion, 5, err)
		}
	}

	if err := u.upgradeV5(repoCfgPath); err != nil {
		return fmt.Errorf("upgrading configuration files from version %d to %d failed: %w", 5, 6, err)
	}

	return nil
}

func (u *CfgUpgrader) upgradeV4() error {
	includesToUpgrade := map[string]struct{}{}
	repoCfgPath := filepath.Join(u.repoRootDir, baur_old.RepositoryCfgFile)

	oldRepoCfg, err := cfg_v4.RepositoryFromFile(repoCfgPath)
	if err != nil {
		return fmt.Errorf("loading repository config %q failed: %w", repoCfgPath, err)
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
		oldInclude, err := cfg_v4.IncludeFromFile(includePath)
		if err != nil {
			return err
		}

		if err := oldInclude.Validate(); err != nil {
			return fmt.Errorf("%s: %w", includePath, err)
		}

		newInclude := v4.UpgradeIncludeConfig(oldInclude)

		if err := fs.BackupFile(includePath); err != nil {
			return err
		}

		if err := newInclude.ToFile(includePath); err != nil {
			return err
		}

		log.Debugf("%s: was updated from v4 to v5", includePath)
	}

	newRepoCfg := v4.UpgradeRepositoryConfig(oldRepoCfg)

	if err := fs.BackupFile(repoCfgPath); err != nil {
		return err
	}

	if err := newRepoCfg.ToFile(repoCfgPath); err != nil {
		return fmt.Errorf("writing new repository config file to disk failed: %w", err)
	}

	log.Debugf("%s: was updated from v4 to v5", repoCfgPath)

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
