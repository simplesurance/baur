package json

import (
	"encoding/json"
	"io"
)

type Mapper interface {
	// AsMap creates a map of an object that contains the fields fields.
	AsMap(fields []string) map[string]any
}

// Encode encodes T as JSON and writes it to w.
func Encode[T ~[]E, E Mapper](w io.Writer, rows T, order []string) error {
	res := make([]map[string]any, 0, len(rows))

	for _, r := range rows {
		res = append(res, r.AsMap(order))
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")

	return enc.Encode(res)
}
