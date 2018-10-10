package postgres

import (
	"database/sql"
	"fmt"

	"github.com/simplesurance/baur/storage"
)

// sqlFieldMap contains a mapping from storage.Fields to table column names
var sqlFieldMap = map[storage.Field]string{
	storage.FieldApplicationName: "application.name",
	storage.FieldBuildDuration:   "duration",
	storage.FieldBuildStartTime:  "build.start_timestamp",
}

// sqlOperatorMap is a mapping from storage.OPs to postgreSQL operator strings
var sqlOperatorMap = map[storage.Op]string{
	storage.OpEQ: "=",
	storage.OpGT: ">",
	storage.OpLT: "<",
}

// sqlOperatorMap is a mapping from storage.OPs to postgreSQL operator strings
var sqlOrderDirectionMap = map[storage.Order]string{
	storage.OrderAsc:  "ASC",
	storage.OrderDesc: "DESC",
}

// RowScanFunc should run rows.Scan and return a value for that row
type RowScanFunc func(rows *sql.Rows) (interface{}, error)

// Query is the sql query struct
type Query struct {
	BaseQuery string
	Filters   []*storage.Filter
	Sorters   []*storage.Sorter
}

func (q *Query) compileFilterStr() (filterStr string, args []interface{}, err error) {
	if len(q.Filters) == 0 {
		return
	}

	filterStr = "WHERE "
	for i, f := range q.Filters {
		field, exist := sqlFieldMap[f.Field]
		if !exist {
			return "", nil, fmt.Errorf("no postgresql mapping for storage field %s exists", f.Field)
		}

		op, exist := sqlOperatorMap[f.Operator]
		if !exist {
			return "", nil, fmt.Errorf("no postgresql mapping for storage operator %s exists", f.Operator)
		}

		filterStr += fmt.Sprintf("%s %s $%d", field, op, i+1)
		args = append(args, f.Value)

		if i+1 < len(q.Filters) {
			filterStr += " AND "
		}
	}

	return
}

func (q *Query) compileSorterStr() (string, error) {
	if len(q.Sorters) == 0 {
		return "", nil
	}

	var sorterStr = "ORDER BY "
	for i, f := range q.Sorters {
		field, exist := sqlFieldMap[f.Field]
		if !exist {
			return "", fmt.Errorf("no postgresql mapping for storage field %s exists", f.Field)
		}

		dir, exist := sqlOrderDirectionMap[f.Order]
		if !exist {
			return "", fmt.Errorf("no postgresql mapping for storage order direction %s exists", f.Order)
		}

		sorterStr += fmt.Sprintf("%s %s", field, dir)

		if i+1 < len(q.Sorters) {
			sorterStr += ",  "
		}
	}

	return sorterStr, nil
}

// Compile compiles the actual sql query
// and returns it along with the query params
func (q *Query) Compile() (query string, args []interface{}, err error) {
	if len(q.Filters) == 0 && len(q.Sorters) == 0 {
		return q.BaseQuery, nil, nil
	}

	filterStr, args, err := q.compileFilterStr()
	if err != nil {
		return "", nil, err
	}

	orderStr, err := q.compileSorterStr()
	if err != nil {
		return "", nil, err
	}

	return fmt.Sprintf("%s %s %s", q.BaseQuery, filterStr, orderStr), args, nil
}
