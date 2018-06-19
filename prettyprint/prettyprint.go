package prettyprint

import (
	"encoding/json"
	"fmt"
)

// AsString returns in as indented JSON
func AsString(in interface{}) string {
	res, err := json.MarshalIndent(in, "", "  ")
	if err != nil {
		return fmt.Sprintf("%+v", in)
	}

	return string(res)
}
