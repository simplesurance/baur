package storage

// CanFilter describer something that can be used as a filter
type CanFilter interface {
	GetField() Field
	GetOperator() SortOperator
	GetValue() interface{}
	IsIncomplete() bool
}
