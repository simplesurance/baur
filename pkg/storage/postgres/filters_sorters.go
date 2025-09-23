package postgres

import (
	"fmt"

	"github.com/simplesurance/baur/v5/pkg/storage"
)

// query assembles an SQL-Query described by storage Filters and Sorters
type query struct {
	BaseQuery string
	Filters   []*storage.Filter
	Sorters   []*storage.Sorter
	Limit     uint
}

func columnName(f storage.Field) (string, error) {
	switch f {
	case storage.FieldApplicationName:
		return "application_name", nil
	case storage.FieldTaskName:
		return "task_name", nil
	case storage.FieldDuration:
		return "duration", nil
	case storage.FieldStartTime:
		return "start_timestamp", nil
	case storage.FieldID:
		return "task_run_id", nil
	case storage.FieldInputString:
		return "input_string_val", nil
	case storage.FieldInputFilePath:
		return "input_file_path", nil

	default:
		return "", fmt.Errorf("no postgresql mapping for storage field %s exists", f)
	}
}

func compileOp(a string, op storage.Op, b string) (string, error) {
	switch op {
	case storage.OpEQ:
		return a + " = " + b, nil
	case storage.OpGT:
		return a + " > " + b, nil
	case storage.OpLT:
		return a + " < " + b, nil
	case storage.OpIN:
		return fmt.Sprintf("%s = ANY (%s)", a, b), nil

	default:
		return "", fmt.Errorf("no postgresql mapping for storage operator %s exists", op)
	}
}

func compileSortOrder(o storage.Order, column string) (string, error) {
	switch o {
	case storage.OrderAsc:
		return column + " ASC", nil
	case storage.OrderDesc:
		return column + " DESC ", nil

	default:
		return "", fmt.Errorf("no postgresql mapping for storage order direction %s exists", o)
	}
}

func (q *query) compileFilterStr() (filterStr string, args []any, err error) {
	if len(q.Filters) == 0 {
		return filterStr, args, err
	}

	for i, f := range q.Filters {
		column, err := columnName(f.Field)
		if err != nil {
			return "", nil, err
		}

		opStr, err := compileOp(column, f.Operator, fmt.Sprintf("$%d", i+1))
		if err != nil {
			return "", nil, err
		}

		filterStr += opStr
		args = append(args, f.Value)

		if i+1 < len(q.Filters) {
			filterStr += " AND "
		}
	}

	return "WHERE " + filterStr, args, err
}

func (q *query) compileSorterStr() (string, error) {
	if len(q.Sorters) == 0 {
		return "", nil
	}

	var sorterStr string
	for i, f := range q.Sorters {
		column, err := columnName(f.Field)
		if err != nil {
			return "", err
		}

		orderStr, err := compileSortOrder(f.Order, column)
		if err != nil {
			return "", err
		}

		sorterStr += orderStr

		if i+1 < len(q.Sorters) {
			sorterStr += ",  "
		}
	}

	return "ORDER BY " + sorterStr, nil
}

func (q *query) compileLimitStr() string {
	if q.Limit == storage.NoLimit {
		return ""
	}

	return fmt.Sprintf("LIMIT %d", q.Limit)
}

// Compile creates the SQL query string and returns it with the arguments for the query
func (q *query) Compile() (query string, args []any, err error) {
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

	limitStr := q.compileLimitStr()

	return fmt.Sprintf("%s %s %s %s", q.BaseQuery, filterStr, orderStr, limitStr), args, nil
}
