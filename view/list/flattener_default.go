package list

import (
	"fmt"
	"github.com/pkg/errors"
	"reflect"
	"strings"
	"text/tabwriter"
)

// DefaultListFlattener provides a default flattener with tabs
var DefaultListFlattener FlattenerFunc = func(l List, hi StringHighlighterFunc, quiet bool) (string, error) {
	var b strings.Builder
	var DLFTabWriter = tabwriter.NewWriter(&b, 0, 0, 8, ' ', 0)

	if len(l.GetData()) == 0 {
		return "", nil
	}

	if !quiet && len(l.columns) > 0 {
		format := getTabwriterFormat(len(l.columns))

		is, err := interfaceSlice(l.GetColumnNames())
		if err != nil {
			return "", errors.Wrap(err, "error converting strings to interfaces")
		}

		fmt.Fprintf(DLFTabWriter, format, is...)
	}

	// write data
	for _, row := range l.GetData() {
		if quiet && len(row) > 0 {
			row = row[:1]
		}

		// todo GetData should store interface{} values instead of strings
		format := getTabwriterFormat(len(row))

		is, err := interfaceSlice(row)
		if err != nil {
			return "", errors.Wrap(err, "error converting strings to interfaces")
		}

		fmt.Fprintf(DLFTabWriter, format, is...)
	}

	err := DLFTabWriter.Flush()
	if err != nil {
		return "", errors.Wrap(err, "couldn't flush the tab writer")
	}

	return b.String(), nil
}

func getTabwriterFormat(inputLen int) string {
	var s []string

	for i := 0; i < inputLen; i++ {
		s = append(s, "%s")
	}

	return strings.Join(s, "\t") + "\n"
}

func interfaceSlice(slice interface{}) ([]interface{}, error) {
	s := reflect.ValueOf(slice)
	if s.Kind() != reflect.Slice {
		return nil, errors.New("InterfaceSlice() given a non-slice type")
	}

	ret := make([]interface{}, s.Len())

	for i := 0; i < s.Len(); i++ {
		ret[i] = s.Index(i).Interface()
	}

	return ret, nil
}
