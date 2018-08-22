package list

// List aggregates all List components
type List struct {
	columns []*Column
	data    [][]string
}

// Column represents a List column
type Column struct {
	Name string
}

// DataProvider provides the data for a List
type DataProvider interface {
	GetData() [][]string
}

// Flattener can display a List
type Flattener interface {
	FlattenList(list List, hi StringHighlighterFunc, quiet bool) (string, error)
}

// NewList instantiates a List
func NewList(cols []*Column, provider DataProvider) *List {
	return &List{
		columns: cols,
		data:    provider.GetData(),
	}
}

// Flatten flattens the List receiver to string
func (l *List) Flatten(viewer Flattener, hi StringHighlighterFunc, quiet bool) (string, error) {
	return viewer.FlattenList(*l, hi, quiet)
}

func (l *List) GetData() [][]string {
	return l.data
}

func (l *List) GetColumnNames() (names []string) {
	for _, col := range l.columns {
		names = append(names, col.Name)
	}

	return
}
