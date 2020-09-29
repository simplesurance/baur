package baur

import (
	"fmt"
	"path/filepath"
	"strings"

	baur_old "github.com/simplesurance/baur"
	cfg_old "github.com/simplesurance/baur/cfg"

	v4 "github.com/simplesurance/baur/v1/cfg/upgrade/v4"
	"github.com/simplesurance/baur/v1/internal/fs"
	"github.com/simplesurance/baur/v1/internal/log"
	"github.com/simplesurance/baur/v1/internal/prettyprint"
)

type CfgUpgrader struct {
	newIncludeID string
	repoRootDir  string
}

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
		if err := newAppCfg.Validate(); err != nil {
			return fmt.Errorf("validation of upgraded app config %q failed: %w\n+%v",
				cfgPath, err, prettyprint.AsString(newAppCfg),
			)
		}

		if err := fs.BackupFile(cfgPath); err != nil {
			return err
		}

		if err := newAppCfg.ToFile(cfgPath); err != nil {
			return err
		}

		log.Debugf("%s: updated", cfgPath)

		for _, includePath := range appCfg.Build.Includes {
			path := strings.Replace(includePath, "$ROOT", repoRootDir, -1)
			if !filepath.IsAbs(path) {
				path = filepath.Join(app.Path, path)
			}

			(*includesToUpgrade)[path] = struct{}{}
		}
	}

	return nil
}

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
