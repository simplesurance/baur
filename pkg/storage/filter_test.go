package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrderFromStr(t *testing.T) {
	tests := []struct {
		in      string
		want    Order
		wantErr bool
	}{
		{in: "asc", want: OrderAsc},
		{in: "desc", want: OrderDesc},
		{in: "ASC", want: OrderAsc},
		{in: "DeSc", want: OrderDesc},
		{in: "", want: SortInvalid, wantErr: true},
		{in: "ascending", want: SortInvalid, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			got, err := OrderFromStr(tc.in)
			assert.Equal(t, tc.want, got)

			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestOrderString(t *testing.T) {
	assert.Equal(t, "asc", OrderAsc.String())
	assert.Equal(t, "desc", OrderDesc.String())
	assert.Equal(t, "invalid", SortInvalid.String())
	assert.Equal(t, "invalid", Order(99).String())
}

func TestSorterString(t *testing.T) {
	s := &Sorter{Field: FieldDuration, Order: OrderDesc}

	assert.Equal(t, "FieldDuration-desc", s.String())
}

func TestFieldString(t *testing.T) {
	tests := map[Field]string{
		FieldApplicationName: "FieldApplicationName",
		FieldTaskName:        "FieldTaskName",
		FieldDuration:        "FieldDuration",
		FieldStartTime:       "FieldStartTime",
		FieldID:              "FieldID",
		FieldInputString:     "FieldInputString",
		FieldUndefined:       "FieldUndefined",
		Field(99):            "FieldUndefined",
	}

	for f, want := range tests {
		assert.Equal(t, want, f.String())
	}
}

func TestOpString(t *testing.T) {
	assert.Equal(t, "OpEQ", OpEQ.String())
	assert.Equal(t, "OpGT", OpGT.String())
	assert.Equal(t, "OpUndefined", Op(99).String())
}

// All defined operators and fields must have a name, they are used in error messages.
func TestStringOfAllDefinedValues(t *testing.T) {
	assert.Equal(t, "OpLT", OpLT.String())
	assert.Equal(t, "OpIN", OpIN.String())
	assert.Equal(t, "FieldInputFilePath", FieldInputFilePath.String())
}
