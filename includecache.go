package baur

import (
	"path/filepath"

	"github.com/simplesurance/baur/cfg"
)

type includeCache struct {
	cache map[string]*cfg.Include
}

func newIncludeCache() *includeCache {
	return &includeCache{cache: map[string]*cfg.Include{}}
}

// load loads an cfg.Include from path.
// If the the include file was already loaded in the past, cfg.Include is
// returned from the cache and not read & parsed again.
func (im *includeCache) load(path string) (*cfg.Include, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	if include, exist := im.cache[path]; exist {
		return include, nil
	}

	include, err := cfg.IncludeFromFile(absPath)
	if err != nil {
		return nil, err
	}

	err = include.Validate()
	if err != nil {
		return nil, err
	}

	im.cache[absPath] = include

	return include, nil
}
