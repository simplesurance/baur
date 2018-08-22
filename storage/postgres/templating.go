package postgres

import (
	"fmt"
	"strings"
)

const PlaceholderTemplate = "{{%s}}"

func stringHasPlaceholder(string, key string) bool {
	return strings.Contains(strings.ToLower(string), WrapKey(key))
}

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
