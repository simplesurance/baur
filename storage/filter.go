package storage

type CanFilter interface {
	GetField() Field
	GetOperator() SortOperator
	GetValue() interface{}
	IsIncomplete() bool
}
