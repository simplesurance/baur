package resolver

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvVarResolve(t *testing.T) {
	const envVar = "_baurTestEnvVar"
	const envVarVal = "hello123"

	testStr := fmt.Sprintf("test {{ env \"%s\" }} {{ env \"%s\" }}bye", envVar, envVar)
	expectedResult := fmt.Sprintf("test %s %sbye", envVarVal, envVarVal)

	resolver := &GoTemplate{}

	os.Setenv(envVar, envVarVal)
	t.Cleanup(func() {
		os.Unsetenv(envVar)
	})

	res, err := resolver.Resolve(testStr)
	require.NoError(t, err)
	require.Equal(t, expectedResult, res)
}

func TestUUIDVar(t *testing.T) {
	r := &GoTemplate{}

	result, err := r.Resolve("{{ uuid }}")
	assert.NoError(t, err)

	matched, err := regexp.MatchString(`(?i)^[0-9A-F]{8}-[0-9A-F]{4}-[4][0-9A-F]{3}-[89AB][0-9A-F]{3}-[0-9A-F]{12}$`, result)
	assert.NoError(t, err)
	assert.True(t, matched)
}
