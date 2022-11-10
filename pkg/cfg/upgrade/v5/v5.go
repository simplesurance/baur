// Package v5 provides helpers to convert baur configs in v5 format (baur 2.x) to v6 (baur 3.0)
package v5

import (
	cfg_v5 "github.com/simplesurance/baur/v2/pkg/cfg"

	"github.com/simplesurance/baur/v3/pkg/cfg"
)

func UpgradeRepositoryConfig(old *cfg_v5.Repository) *cfg.Repository {
	return &cfg.Repository{
		ConfigVersion: cfg.Version,
		Database: cfg.Database{
			PGSQLURL: old.Database.PGSQLURL,
		},
		Discover: cfg.Discover{
			Dirs:        old.Discover.Dirs,
			SearchDepth: old.Discover.SearchDepth,
		},
	}
}
