package validation

import (
	"errors"
	"fmt"
	"unicode"
)

// StrID ensures that id does not contain leading or trailing white spaces
// ([unicode.IsSpace] and only printable characters ([unicode.IsPrint].
func StrID(id string) error {
	for pos, r := range id {
		if (pos == 0 || pos == len(id)-1) && unicode.IsSpace(r) {
			return errors.New("contains leading or trailing white spaces")
		}

		if !unicode.IsPrint(r) {
			return fmt.Errorf("contains non-printable character: %+q", r)
		}
	}

	return nil
}
