package cfg

import (
	"fmt"
	"strings"
)

type Environment struct {
	Variables []string `toml:"variables" comment:"environment variables, in the format KEY=VALUE, that are set when the command is executed.\n The variables and their values are tracked automatically as inputs."`
}

func (e *Environment) validate() error {
	for _, v := range e.Variables {
		if !strings.ContainsRune(v, '=') {
			return newFieldError(fmt.Sprintf("'=' missing in %q, environment variables must be defined in the format NAME=VALUE", v), "variables")
		}
	}

	return nil
}
