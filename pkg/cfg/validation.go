package cfg

import (
	"errors"
	"fmt"
	"strings"
)

var forbiddenNameRunes = [...]rune{
	'.',
	'*',
	'#',
}

func validateTaskOrAppName(name string) error {
	if len(name) == 0 {
		return errors.New("can not be empty")
	}

	for _, r := range forbiddenNameRunes {
		if strings.ContainsRune(name, r) {
			return fmt.Errorf("'%c' character not allowed in name", r)
		}
	}

	return nil
}
