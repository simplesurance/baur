package postgres

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/simplesurance/baur/storage"
)

// Filter is an implementation of CanFilter
type Filter struct {
	field    storage.Field
	operator storage.SortOperator
	value    interface{}
}

// IsIncomplete implementing CanFilter
func (f *Filter) IsIncomplete() bool {
	return f.GetField() == storage.FieldNull || f.GetOperator() == storage.OperatorNull || f.GetValue() == nil
}

// GetField implementing CanFilter
func (f *Filter) GetField() storage.Field {
	return f.field
}

// GetOperator implementing CanFilter
func (f *Filter) GetOperator() storage.SortOperator {
	return f.operator
}

// GetValue implementing CanFilter
func (f *Filter) GetValue() interface{} {
	return f.value
}

// NewFilter is the filter constructor
func NewFilter(field storage.Field, operator storage.SortOperator, value interface{}) *Filter {
	return &Filter{field, operator, value}
}

// Filters provides a collection of filters next to the SQLMap
type Filters struct {
	filters []*Filter
	sqlMap  SQLStringer
}

var (
	// TplFilterGlue is the glue between filters
	TplFilterGlue = " AND "
	// PlaceholderFilters is the filters placeholder key
	PlaceholderFilters = "filters"
)

// SetFilters sets filters on a query
func (q *Query) SetFilters(filters []storage.CanFilter) error {
	for _, filter := range filters {
		if filter.IsIncomplete() {
			return errors.New("incomplete filter")
		}
	}

	if !stringHasPlaceholder(q.baseQuery, PlaceholderFilters) {
		return fmt.Errorf("the %s placeholder was not found in query", WrapKey(PlaceholderFilters))
	}

	if !strings.Contains(strings.ToLower(q.baseQuery), "where") {
		return errors.New("you're trying to set filters on a query that lacks a WHERE clause")
	}

	q.filters = q.getFiltersFromCanFilterSlice(filters)
	q.filters.sqlMap = q.sqlMap

	return nil
}

func (q *Query) getFiltersFromCanFilterSlice(canFilters []storage.CanFilter) (filters Filters) {
	f := filters.filters
	for _, cf := range canFilters {
		f = append(f, NewFilter(cf.GetField(), cf.GetOperator(), cf.GetValue()))
	}
	filters.filters = f

	return
}

// String returns the string representation of a filters collection
func (f Filters) String() string {
	var pieces []string

	for i, filter := range f.filters {
		// something = $1
		fieldName, err := filter.GetField().GetName(f.sqlMap.GetFieldsMap())
		if err != nil {
			panic("undefined field")
		}

		operatorName, err := filter.GetOperator().GetName(f.sqlMap.GetOperatorsMap())
		if err != nil {
			panic("undefined operator")
		}

		piece := fmt.Sprintf("%s %s $%d", sqlQuote(fieldName), operatorName, i+1)

		pieces = append(pieces, piece)
	}

	return strings.Join(pieces, TplFilterGlue)
}

// GetValues returns the values of a filters collection
func (f Filters) GetValues() []interface{} {
	var values []interface{}

	for _, filter := range f.filters {
		values = append(values, filter.GetValue())
	}

	return values
}

// Compile looks for the filters placeholder and returns the query
// with the WHERE conditions included, along with the query params.
// Only replaces 1 occurrence. Returns error if the placeholder is not found.
func (f Filters) Compile(queryTpl string, mapper SQLStringer) (string, []interface{}, error) {
	if len(f.filters) == 0 {
		if stringHasPlaceholder(queryTpl, PlaceholderFilters) {
			return "", nil, errors.New("tpl contains the filters placeholder, but query has no filters")
		}

		return queryTpl, nil, nil
	}

	compiledQuery, err := setPlaceholderValue(PlaceholderFilters, queryTpl, f.String(), 1)
	if err != nil {
		return "", nil, errors.Wrap(err, "couldn't replace filters placeholder")
	}

	return compiledQuery, f.GetValues(), nil
}
