package postgres

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v5/pkg/storage"
)

// normWS collapses all whitespace runs to a single space.
func normWS(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func TestColumnName(t *testing.T) {
	tests := map[storage.Field]string{
		storage.FieldApplicationName: "application_name",
		storage.FieldTaskName:        "task_name",
		storage.FieldDuration:        "duration",
		storage.FieldStartTime:       "start_timestamp",
		storage.FieldID:              "task_run_id",
		storage.FieldInputString:     "input_string_val",
		storage.FieldInputFilePath:   "input_file_path",
	}

	for f, want := range tests {
		got, err := columnName(f)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	}

	_, err := columnName(storage.FieldUndefined)
	require.Error(t, err)
}

func TestCompileOp(t *testing.T) {
	tests := map[storage.Op]string{
		storage.OpEQ: "col = $1",
		storage.OpGT: "col > $1",
		storage.OpLT: "col < $1",
		storage.OpIN: "col = ANY ($1)",
	}

	for op, want := range tests {
		got, err := compileOp("col", op, "$1")
		require.NoError(t, err)
		assert.Equal(t, want, got)
	}

	_, err := compileOp("col", storage.Op(99), "$1")
	require.Error(t, err)
}

func TestCompileSortOrder(t *testing.T) {
	got, err := compileSortOrder(storage.OrderAsc, "col")
	require.NoError(t, err)
	assert.Equal(t, "col ASC", normWS(got))

	got, err = compileSortOrder(storage.OrderDesc, "col")
	require.NoError(t, err)
	assert.Equal(t, "col DESC", normWS(got))

	_, err = compileSortOrder(storage.SortInvalid, "col")
	require.Error(t, err)
}

func TestQueryCompile(t *testing.T) {
	const base = "SELECT 1 FROM t"

	t.Run("without filters and sorters returns the base query", func(t *testing.T) {
		q := query{BaseQuery: base}

		got, args, err := q.Compile()
		require.NoError(t, err)
		assert.Equal(t, base, got)
		assert.Empty(t, args)
	})

	t.Run("filters are joined with AND and use numbered args in order", func(t *testing.T) {
		q := query{
			BaseQuery: base,
			Filters: []*storage.Filter{
				{Field: storage.FieldApplicationName, Operator: storage.OpEQ, Value: "app"},
				{Field: storage.FieldID, Operator: storage.OpGT, Value: 5},
				{Field: storage.FieldTaskName, Operator: storage.OpIN, Value: []string{"a", "b"}},
			},
		}

		got, args, err := q.Compile()
		require.NoError(t, err)

		assert.Equal(t,
			base+" WHERE application_name = $1 AND task_run_id > $2 AND task_name = ANY ($3)",
			normWS(got),
		)
		assert.Equal(t, []any{"app", 5, []string{"a", "b"}}, args)
	})

	t.Run("sorters are comma separated, in the specified order", func(t *testing.T) {
		q := query{
			BaseQuery: base,
			Sorters: []*storage.Sorter{
				{Field: storage.FieldDuration, Order: storage.OrderDesc},
				{Field: storage.FieldStartTime, Order: storage.OrderAsc},
			},
		}

		got, args, err := q.Compile()
		require.NoError(t, err)

		assert.Equal(t, base+" ORDER BY duration DESC , start_timestamp ASC", normWS(got))
		assert.Empty(t, args)
	})

	t.Run("filter, sorter and limit", func(t *testing.T) {
		q := query{
			BaseQuery: base,
			Filters:   []*storage.Filter{{Field: storage.FieldTaskName, Operator: storage.OpLT, Value: "x"}},
			Sorters:   []*storage.Sorter{{Field: storage.FieldID, Order: storage.OrderAsc}},
			Limit:     7,
		}

		got, args, err := q.Compile()
		require.NoError(t, err)

		assert.Equal(t, base+" WHERE task_name < $1 ORDER BY task_run_id ASC LIMIT 7", normWS(got))
		assert.Equal(t, []any{"x"}, args)
	})

	t.Run("NoLimit adds no LIMIT clause", func(t *testing.T) {
		q := query{
			BaseQuery: base,
			Sorters:   []*storage.Sorter{{Field: storage.FieldID, Order: storage.OrderAsc}},
			Limit:     storage.NoLimit,
		}

		got, _, err := q.Compile()
		require.NoError(t, err)
		assert.NotContains(t, got, "LIMIT")
	})

	// Limit is documented as applying to all results of Storer.TaskRuns().
	t.Run("limit is applied without filters and sorters", func(t *testing.T) {
		q := query{BaseQuery: base, Limit: 3}

		got, args, err := q.Compile()
		require.NoError(t, err)
		assert.Contains(t, got, "LIMIT 3")
		assert.True(t, strings.HasPrefix(got, base))
		assert.Empty(t, args)
	})

	t.Run("errors", func(t *testing.T) {
		tests := map[string]query{
			"unknown filter field": {
				BaseQuery: base,
				Filters:   []*storage.Filter{{Field: storage.FieldUndefined, Operator: storage.OpEQ}},
			},
			"unknown filter operator": {
				BaseQuery: base,
				Filters:   []*storage.Filter{{Field: storage.FieldID, Operator: storage.Op(99)}},
			},
			"unknown sorter field": {
				BaseQuery: base,
				Sorters:   []*storage.Sorter{{Field: storage.FieldUndefined, Order: storage.OrderAsc}},
			},
			"invalid sort order": {
				BaseQuery: base,
				Sorters:   []*storage.Sorter{{Field: storage.FieldID, Order: storage.SortInvalid}},
			},
		}

		for name, q := range tests {
			t.Run(name, func(t *testing.T) {
				_, _, err := q.Compile()
				require.Error(t, err)
			})
		}
	})
}
