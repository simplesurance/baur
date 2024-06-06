package baur

import "fmt"

type ErrDuplicateAppNames struct {
	AppName  string
	AppPath1 string
	AppPath2 string
}

func (e *ErrDuplicateAppNames) Error() string {
	return fmt.Sprintf(
		"app names must be unique but the following app configs use the same app name %q: %s, %s",
		e.AppName, e.AppPath1, e.AppPath2,
	)
}
