package resolver

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEnvVarResolve(t *testing.T) {
	const envVar = "_baurTestEnvVar"
	const envVarVal = "hello123"

	testStr := fmt.Sprintf("test {{ env %s }} {{ env %s }}bye", envVar, envVar)
	expectedResult := fmt.Sprintf("test %s %sbye", envVarVal, envVarVal)

	resolver := &EnvVar{}

	os.Setenv(envVar, envVarVal)
	t.Cleanup(func() {
		os.Unsetenv(envVar)
	})

	res, err := resolver.Resolve(testStr)
	require.NoError(t, err)
	require.Equal(t, expectedResult, res)
}
