package dataprovider

// MemoryDataProviderFunc is a memory data provider (for tests)
type MemoryDataProviderFunc func() [][]string

// GetData implements the provider interface
func (f MemoryDataProviderFunc) GetData() [][]string {
	return f()
}
