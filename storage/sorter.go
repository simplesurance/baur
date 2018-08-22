package storage

import "fmt"

type OrderDirection string

type CanSort interface {
	GetField() Field
	GetDirection() OrderDirection
	IsIncomplete() bool
}

const (
	OrderAsc  OrderDirection = "ASC"
	OrderDesc OrderDirection = "DESC"
)

type SortOperator int

const (
	OperatorNull SortOperator = iota
	OperatorEq
	OperatorLt
	OperatorGt
	OperatorLte
	OperatorGte
)

func (o SortOperator) GetName(operators map[SortOperator]string) (string, error) {
	if name, ok := operators[o]; ok {
		return name, nil
	}

	return "", fmt.Errorf("operator %d missing from collection", o)
}
