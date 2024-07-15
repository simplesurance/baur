// Package v5 provides helpers to convert baur configs in v5 format (baur 2.x) to v7 (baur 4.x)
package v5

import (
	"github.com/simplesurance/baur/v5/pkg/cfg"
)

func UpgradeRepositoryConfig(old *cfg.Repository) *cfg.Repository {
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
