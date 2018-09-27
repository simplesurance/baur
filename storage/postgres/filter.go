package postgres

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/simplesurance/baur/storage"
)

type filter struct {
	field    storage.Field
	operator storage.SortOperator
	value    interface{}
}

func (f *filter) IsIncomplete() bool {
	return f.GetField() == storage.FieldNull || f.GetOperator() == storage.OperatorNull || f.GetValue() == nil
}

func (f *filter) GetField() storage.Field {
	return f.field
}

func (f *filter) GetOperator() storage.SortOperator {
	return f.operator
}

func (f *filter) GetValue() interface{} {
	return f.value
}

func NewFilter(field storage.Field, operator storage.SortOperator, value interface{}) *filter {
	return &filter{field, operator, value}
}

type Filters struct {
	filters []*filter
	sqlMap  SqlStringer
}

var (
	TplFilterGlue      = " AND "
	PlaceholderFilters = "filters"
)

func (q *Query) SetFilters(filters []storage.CanFilter) error {
	for _, filter := range filters {
		if filter.IsIncomplete() {
			return errors.New("incomplete filter")
		}
	}

	if !stringHasPlaceholder(q.baseQuery, PlaceholderFilters) {
		return errors.New(fmt.Sprintf(
			"the %s placeholder was not found in query",
			WrapKey(PlaceholderFilters),
		))
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

func (f Filters) GetValues() []interface{} {
	var values []interface{}

	for _, filter := range f.filters {
		values = append(values, filter.GetValue())
	}

	return values
}

// Compile looks for the filters placeholder and returns the query
// with the WHERE conditions included, along with the query params.
// Only replaces 1 occurence. Returns error if the placeholder is not found.
func (f Filters) Compile(queryTpl string, mapper SqlStringer) (string, []interface{}, error) {
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
