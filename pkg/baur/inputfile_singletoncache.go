package baur

import (
	"github.com/simplesurance/baur/v3/internal/digest"
)

const inputFileSingletonCacheInitialSize = 250

type FileHashFn func(path string) (*digest.Digest, error)

// InputFileSingletonCache stores previously created InputFiles and returns
// them for the same path instead of creating another instance.
type InputFileSingletonCache struct {
	cache      map[string]*InputFile
	fileHasher FileHashFn
}

// newInputFile SingletonCache creates a inputFileSingletonCache.
func NewInputFileSingletonCache(fileHasher FileHashFn) *InputFileSingletonCache {
	return &InputFileSingletonCache{
		cache:      make(map[string]*InputFile, inputFileSingletonCacheInitialSize),
		fileHasher: fileHasher,
	}
}

// CreateOrGetInputFile returns a new InputFile if none with the same
// repoRootPath and relPath has been created before with this method.
// Otherwise it returns a reference to the previously created InputFile.
func (c *InputFileSingletonCache) CreateOrGetInputFile(absPath, relPath string) *InputFile {
	if f, exists := c.cache[absPath]; exists {
		return f
	}

	f := NewInputFile(absPath, relPath, c.fileHasher)
	c.cache[absPath] = f

	return f
}
