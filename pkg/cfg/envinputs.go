package cfg

import (
	"fmt"
	"strings"
)

type EnvVarsInputs struct {
	Names    []string `toml:"names" comment:"Names of environment variables that are tracked.\n Glob patterns are supported, all names are case-sensitive.\n Declared but undefined environment variable are treated as not existing."`
	Optional bool     `toml:"optional" comment:"When optional is true, a variable pattern matching 0 defined variables will not cause an error."`
}

// Validate always returns nil.
func (ei *EnvVarsInputs) Validate() error {
	// unix and windows OSes agree on `=` being an invalid env name
	// characters, apart from that Windows allows almost any character,
	// Linux only a small set.
	// The validation is kept to a minimum, invalid names will later not
	// match any defined vars.
	for _, e := range ei.Names {
		if len(e) == 0 {
			return newFieldError("element can not be empty", "variables")
		}

		if strings.ContainsRune(e, '=') {
			return newFieldError(
				fmt.Sprintf("environment variable name %q contains invalid character '='", e),
				"variables",
			)
		}
	}

	return nil
}
