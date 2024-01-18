package prettyprint

import (
	"encoding/json"
	"fmt"
	"strings"
)

// AsString returns in as indented JSON
func AsString(in any) string {
	res, err := json.MarshalIndent(in, "", "  ")
	if err != nil {
		return fmt.Sprintf("%+v", in)
	}

	return string(res)
}

// TruncatedStrSlice returns sl as string, joined by ", ".
// If sl has more then maxElems, only the first maxElems elements will be
// returned and additional truncation marker.
func TruncatedStrSlice(sl []string, maxElems int) string {
	if len(sl) <= maxElems {
		return strings.Join(sl, ", ")
	}

	return strings.Join(sl[:maxElems], ", ") + ", [...]"
}
