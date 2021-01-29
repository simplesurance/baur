package baur

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/golang/groupcache/lru"

	"github.com/simplesurance/baur/v1/pkg/cfg"
)

const inputResolverCacheMaxEntries = 512

type inputResolverCache struct {
	cache *lru.Cache

	hits int
	miss int

	mu sync.Mutex
}

type inputResolverCacheStats struct {
	Entries int
	Hits    int
	Miss    int
}

func newInputResolverCache() *inputResolverCache {
	return &inputResolverCache{
		cache: lru.New(inputResolverCacheMaxEntries),
	}
}

func strSliceStr(in []string) string {
	if len(in) == 0 {
		return "[]"
	}

	return strings.Join(in, ",")
}

func (i *inputResolverCache) goSourcesKey(appdir string, cfg *cfg.GolangSources) string {
	var key strings.Builder

	key.WriteString(appdir)
	key.WriteString(strSliceStr(cfg.Queries))
	key.WriteString(strSliceStr(cfg.Environment))
	key.WriteString(strSliceStr(cfg.BuildFlags))
	key.WriteString(strconv.FormatBool(cfg.Tests))

	return key.String()
}

func (i *inputResolverCache) get(key string) []string {
	i.mu.Lock()
	defer i.mu.Unlock()

	result, exists := i.cache.Get(key)

	if exists {
		i.hits++
	} else {
		i.miss++

		return nil
	}

	if result, ok := result.([]string); ok {
		return result
	}

	panic(fmt.Sprintf("inputResolverCache returned val of type %t", result))
}

func (i *inputResolverCache) set(k string, v []string) {
	i.mu.Lock()
	i.cache.Add(k, v)
	i.mu.Unlock()
}

func (i *inputResolverCache) AddGolangSources(appdir string, gs *cfg.GolangSources, result []string) {
	i.set(i.goSourcesKey(appdir, gs), result)
}

func (i *inputResolverCache) GetGolangSources(appdir string, gs *cfg.GolangSources) []string {
	key := i.goSourcesKey(appdir, gs)
	return i.get(key)
}

func (i *inputResolverCache) fileInputsKey(appdir string, fi *cfg.FileInputs) string {
	var key strings.Builder

	key.WriteString(appdir)
	key.WriteString(strSliceStr(fi.Paths))
	key.WriteString(strconv.FormatBool(fi.Optional))
	key.WriteString(strconv.FormatBool(fi.GitTrackedOnly))

	return key.String()
}

func (i *inputResolverCache) AddFileInputs(appdir string, fi *cfg.FileInputs, result []string) {
	i.set(i.fileInputsKey(appdir, fi), result)
}

func (i *inputResolverCache) GetFileInputs(appdir string, fi *cfg.FileInputs) []string {
	key := i.fileInputsKey(appdir, fi)

	return i.get(key)
}

func (i *inputResolverCacheStats) HitRatio() float64 {
	if i.Hits == 0 && i.Miss == 0 {
		return 0
	}

	return (float64(i.Hits) / float64(i.Hits+i.Miss)) * 100
}

func (i *inputResolverCache) Statistics() *inputResolverCacheStats {
	i.mu.Lock()
	defer i.mu.Unlock()

	return &inputResolverCacheStats{
		Entries: i.cache.Len(),
		Hits:    i.hits,
		Miss:    i.miss,
	}
}
