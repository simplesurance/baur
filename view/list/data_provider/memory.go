package data_provider

type MemoryDataProviderFunc func() [][]string

func (f MemoryDataProviderFunc) GetData() [][]string {
	return f()
}
