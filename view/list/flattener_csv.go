package list

import (
	"bytes"
	"encoding/csv"
	"github.com/pkg/errors"
)

// CsvListFlattener flattens to CSV
var CsvListFlattener FlattenerFunc = func(l List, hi StringHighlighterFunc, quiet bool) (string, error) {
	var buffer bytes.Buffer
	w := csv.NewWriter(&buffer)

	data := l.GetData()

	if quiet {
		for i, row := range data {
			if len(row) > 0 {
				data[i] = row[:1]
			}
		}
	}

	err := w.WriteAll(data)
	if err != nil {
		return "", errors.Wrap(err, "couldn't write data to csv buffer")
	}

	return buffer.String(), nil
}
