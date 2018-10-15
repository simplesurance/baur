package storage

import "fmt"

// OrderDirection represents the ordering direction (asc / desc)
type OrderDirection string

// CanSort is the sorter interface
type CanSort interface {
	GetField() Field
	GetDirection() OrderDirection
	IsIncomplete() bool
}

const (
	// OrderAsc order asc
	OrderAsc OrderDirection = "ASC"
	// OrderDesc order desc
	OrderDesc OrderDirection = "DESC"
)

// SortOperator generic sorting operators
type SortOperator int

const (
	// OperatorNull is the null value
	OperatorNull SortOperator = iota
	// OperatorEq =
	OperatorEq
	// OperatorLt <
	OperatorLt
	// OperatorGt >
	OperatorGt
	// OperatorLte <=
	OperatorLte
	// OperatorGte >=
	OperatorGte
)

// GetName returns the name of the operator
func (o SortOperator) GetName(operators map[SortOperator]string) (string, error) {
	if name, ok := operators[o]; ok {
		return name, nil
	}

	return "", fmt.Errorf("operator %d missing from collection", o)
}
