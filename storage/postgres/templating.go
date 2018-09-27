package postgres

import (
	"fmt"
	"strings"
)

// PlaceholderTemplate is the template for a sql placeholder
const PlaceholderTemplate = "{{%s}}"

func stringHasPlaceholder(string, key string) bool {
	return strings.Contains(strings.ToLower(string), WrapKey(key))
}

// WrapKey wraps a key name into a template placeholder
func WrapKey(key string) string {
	return fmt.Sprintf(PlaceholderTemplate, key)
}

func setPlaceholderValue(key, string, value string, n int) (string, error) {
	if !stringHasPlaceholder(string, key) {
		return "",
			fmt.Errorf(`string "%s" does not contain missing key `+PlaceholderTemplate, string, key)
	}

	return strings.Replace(string, WrapKey(key), value, n), nil
}
