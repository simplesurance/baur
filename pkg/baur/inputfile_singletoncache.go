package baur

import (
	"github.com/simplesurance/baur/v3/internal/digest"
)

const inputFileSingletonCacheInitialSize = 250

type FileHashFn func(path string) (*digest.Digest, error)

// InputFileSingletonCache stores previously created Inputs and returns
// them for the same path instead of creating another instance.
type InputFileSingletonCache struct {
	cache map[string]*InputFile
}

// newInputFile SingletonCache creates a inputFileSingletonCache.
func NewInputFileSingletonCache() *InputFileSingletonCache {
	return &InputFileSingletonCache{
		cache: make(map[string]*InputFile, inputFileSingletonCacheInitialSize),
	}
}

func (c *InputFileSingletonCache) Get(absPath string) (f *InputFile, exists bool) {
	f, exists = c.cache[absPath]
	return f, exists
}

// Add adds f to the cache and returns it.
func (c *InputFileSingletonCache) Add(f *InputFile) *InputFile {
	c.cache[f.AbsPath()] = f
	return f
}
