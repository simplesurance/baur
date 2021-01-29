package resolver

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

type EnvVar struct{}

var envVarRe = regexp.MustCompile(`{{ env ([\w-]+) }}`)

func (*EnvVar) Resolve(in string) (string, error) {
	matches := envVarRe.FindAllStringSubmatch(in, -1)
	for _, m := range matches {
		if len(m) != 2 {
			return "", fmt.Errorf("invalid cfg env variable: '%v'", m)
		}

		fullVar := m[0]
		envVarName := m[1]

		envVal, exist := os.LookupEnv(envVarName)
		if !exist {
			return "", fmt.Errorf("environment variable %q is referenced by %q but is undefined", fullVar, envVarName)
		}

		in = strings.Replace(in, fullVar, envVal, 1)
	}

	return in, nil
}
