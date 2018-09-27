package postgres

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/simplesurance/baur/storage"
)

var (
	PlaceholderSorters = "sorters"
)

type Sorter struct {
	field     storage.Field
	direction storage.OrderDirection
}

func (s *Sorter) IsIncomplete() bool {
	return s.GetField() == storage.FieldNull || s.GetDirection() == ""
}

func (s *Sorter) GetField() storage.Field {
	return s.field
}

func (s *Sorter) GetDirection() storage.OrderDirection {
	return s.direction
}

func NewSorter(field storage.Field, direction storage.OrderDirection) *Sorter {
	return &Sorter{field, direction}
}

type Sorters struct {
	sorters []*Sorter
	sqlMap  SqlStringer
}

func (q *Query) SetSorters(sorters []storage.CanSort) error {
	if !stringHasPlaceholder(q.baseQuery, PlaceholderSorters) {
		return errors.New(fmt.Sprintf(
			"the %s placeholder was not found in query",
			WrapKey(PlaceholderSorters),
		))
	}

	if !strings.Contains(strings.ToLower(q.baseQuery), "order by") {
		return errors.New("you're trying to set sorters on a query with no order by clause")
	}

	q.sorters = getSortersFromCanSortSlice(sorters)
	q.sorters.sqlMap = q.sqlMap

	return nil
}

func getSortersFromCanSortSlice(canSorters []storage.CanSort) (sorters Sorters) {
	var s []*Sorter

	for _, cs := range canSorters {
		s = append(s, NewSorter(cs.GetField(), cs.GetDirection()))
	}

	sorters.sorters = s

	return
}

func (s Sorters) String() string {
	if len(s.sorters) == 0 {
		return ""
	}

	var pieces []string

	for _, sort := range s.sorters {
		fieldName, err := sort.GetField().GetName(s.sqlMap.GetFieldsMap())
		if err != nil {
			panic("undefined field")
		}
		pieces = append(pieces, fmt.Sprintf("%s %s", sqlQuote(fieldName), sort.GetDirection()))
	}

	return strings.Join(pieces, ", ")
}

// Compile looks for the sorters placeholder and returns the query with the SORT included.
// Only replaces 1 occurence.
// Returns error if the placeholder is not found.
func (s Sorters) Compile(queryTpl string, mapper SqlStringer) (string, error) {
	if len(s.sorters) == 0 {
		if stringHasPlaceholder(queryTpl, PlaceholderSorters) {
			return "", errors.New("tpl contains the filters placeholder, but query has no filters")
		}

		return queryTpl, nil
	}
	return setPlaceholderValue(PlaceholderSorters, queryTpl, s.String(), 1)
}
