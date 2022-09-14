package baur

import "path/filepath"

const inputFileSingletonCacheInitialSize = 250

// InputFileSingletonCache stores previously created InputFiles and returns
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

// CreateOrGetInputFile returns a new InputFile if none with the same
// repoRootPath and relPath has been created before with this method.
// Otherwise it returns a reference to the previously created InputFile.
func (c *InputFileSingletonCache) CreateOrGetInputFile(repoRootPath, relPath string) *InputFile {
	absPath := filepath.Join(repoRootPath, relPath)

	if f, exists := c.cache[absPath]; exists {
		return f
	}

	f := NewInputFile(repoRootPath, relPath)
	c.cache[absPath] = f

	return f
}
