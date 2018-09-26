package postgres

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

// RowScanFunc should run rows.Scan and return a value for that row
type RowScanFunc func(rows *sql.Rows) (interface{}, error)

// Query is the sql query struct
type Query struct {
	baseQuery string
	filters   Filters
	sorters   Sorters

	sqlMap SQLStringer
}

// NewQuery is the Query constructor
func NewQuery(baseQuery string, sqlMap SQLMap) *Query {
	return &Query{
		baseQuery: baseQuery,
		sqlMap:    sqlMap,
	}
}

// Compile compiles the actual sql query
// and returns it along with the query params
func (q *Query) Compile() (compiledQuery string, params []interface{}, err error) {
	compiledQuery, params, err = q.filters.Compile(q.baseQuery, q)
	if err != nil {
		return "", nil, errors.Wrap(err, "couldn't compile filters")
	}

	compiledQuery, err = q.sorters.Compile(compiledQuery, q)
	if err != nil {
		return "", nil, errors.Wrap(err, "couldn't compile sorters")
	}

	return
}

// GetFieldsMap is part of the SQLStringer implementation in Query
func (q *Query) GetFieldsMap() SQLFields {
	return q.sqlMap.GetFieldsMap()
}

// GetOperatorsMap is part of the SQLStringer implementation in Query
func (q *Query) GetOperatorsMap() SQLFilterOperators {
	return q.sqlMap.GetOperatorsMap()
}

func sqlQuote(subject string) string {
	if !strings.Contains(subject, " ") {
		return subject
	}
	return fmt.Sprintf("'%s'", subject)
}

// RunSelectQuery runs a sql_query and extracts the results using the row scanner func
func RunSelectQuery(c Client, query Query, rowScanFunc RowScanFunc) ([]interface{}, error) {
	compiledQuery, params, err := query.Compile()
	if err != nil {
		return nil, errors.Wrap(err, "error while trying to compile the query")
	}

	rows, err := c.Db.Query(compiledQuery, params...)
	if err != nil {
		return nil, errors.Wrapf(err, "db query failed: \"%v\" (params %q) ", compiledQuery, params)
	}

	var results []interface{}
	for rows.Next() {
		convertedRow, err := rowScanFunc(rows)
		if err != nil {
			rows.Close()
			return nil,
				errors.Wrapf(err, "parsing result of query %q (params %q) failed", compiledQuery, params)
		}

		results = append(results, convertedRow)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrapf(err, "iterating over rows failed (query %q, params %q)", compiledQuery, params)
	}

	return results, nil
}
