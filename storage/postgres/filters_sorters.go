package postgres

import (
	"fmt"

	"github.com/simplesurance/baur/storage"
)

// sqlFieldMap contains a mapping from storage.Fields to table column names
var sqlFieldMap = map[storage.Field]string{
	storage.FieldApplicationName: "application_name",
	storage.FieldTaskName:        "task_name",
	storage.FieldDuration:        "duration",
	storage.FieldStartTime:       "start_timestamp",
	storage.FieldID:              "task_run_id",
}

// sqlOperatorMap is a mapping from storage.OPs to postgreSQL operator strings
var sqlOperatorMap = map[storage.Op]string{
	storage.OpEQ: "=",
	storage.OpGT: ">",
	storage.OpLT: "<",
	storage.OpIN: "= ANY",
}

// sqlOperatorMap is a mapping from storage.OPs to postgreSQL operator strings
var sqlOrderDirectionMap = map[storage.Order]string{
	storage.OrderAsc:  "ASC",
	storage.OrderDesc: "DESC",
}

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

		// parenthesis around $%d are needed for the ANY query, the
		// syntax is also valid for all other supported filters
		filterStr += fmt.Sprintf("%s %s ($%d)", field, op, i+1)
		args = append(args, f.Value)

		if i+1 < len(q.Filters) {
			filterStr += " AND "
		}
	}

	return filterStr, args, err
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

// Compile creates the SQL query string and returns it with the arguments for the query
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
