package storage

import (
	"errors"
	"fmt"
	"strings"
)

// Field represents data fields that can be used in sort and filter operations
type Field int

// Defines the available data fields
const (
	FieldUndefined Field = iota
	FieldApplicationName
	FieldTaskName
	FieldDuration
	FieldStartTime
	FieldID
	FieldInputString
	FieldInputFilePath
)

func (f Field) String() string {
	switch f {
	case FieldApplicationName:
		return "FieldApplicationName"
	case FieldTaskName:
		return "FieldTaskName"
	case FieldDuration:
		return "FieldDuration"
	case FieldStartTime:
		return "FieldStartTime"
	case FieldID:
		return "FieldID"
	case FieldInputString:
		return "FieldInputString"
	default:
		return "FieldUndefined"
	}
}

// Filter specifies filter operatons for queries
type Filter struct {
	Field    Field
	Operator Op
	Value    interface{}
}

// Op describes the filter operator
type Op int

const (
	// OpEQ represents an equal (=) operator
	OpEQ Op = iota
	// OpGT represents a greater than (>) operator
	OpGT
	// OpLT represents a smaller than (<) operator
	OpLT
	// OpIN represents a In operator, works like the SQL IN operator, the
	// corresponding Value field in The filter struct must be a slice
	OpIN
)

func (o Op) String() string {
	switch o {
	case OpEQ:
		return "OpEQ"
	case OpGT:
		return "OpGT"
	default:
		return "OpUndefined"
	}
}

// Order specifies the sort order
type Order int

const (
	// SortInvalid represents an invalid sort value
	SortInvalid Order = iota
	// OrderAsc sorts ascending
	OrderAsc
	// OrderDesc sorts descending
	OrderDesc
)

func (s Order) String() string {
	switch s {
	case OrderAsc:
		return "asc"
	case OrderDesc:
		return "desc"
	default:
		return "invalid"
	}
}

//OrderFromStr converts a string to an Order
func OrderFromStr(s string) (Order, error) {
	switch strings.ToLower(s) {
	case "asc":
		return OrderAsc, nil
	case "desc":
		return OrderDesc, nil
	default:
		return SortInvalid, errors.New("undefined order")
	}
}

// Sorter specifies how the result of queries should be sorted
type Sorter struct {
	Field Field
	Order Order
}

// String return the string representation
func (s *Sorter) String() string {
	return fmt.Sprintf("%s-%s", s.Field, s.Order)
}
