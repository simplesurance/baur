package resolver

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplatingWithUndefinedVarFails(t *testing.T) {
	templ := NewGoTemplate("myapp", "/", func() (string, error) { return "", nil })

	res, err := templ.Resolve("{{ .myvar }}")
	require.Error(t, err)
	require.Empty(t, res)
}

func TestResolve(t *testing.T) {
	const envVar = "_baurTestEnvVar"
	const envVarVal = "hello123"
	const rootDir = "/tmp/f00bar"
	const commitID = "commit1231231231"
	const appName = "my-app-name"

	os.Setenv(envVar, envVarVal)
	t.Cleanup(func() {
		os.Unsetenv(envVar)
	})

	testCases := []struct {
		name           string
		input          string
		expectedResult string
		validator      func(string) error
	}{
		{
			name:           "Test env",
			input:          fmt.Sprintf("test {{ env \"%s\" }} {{ env \"%s\" }}bye", envVar, envVar),
			expectedResult: fmt.Sprintf("test %s %sbye", envVarVal, envVarVal),
		},
		{
			name:           "Test root",
			input:          "{{ .Root }}",
			expectedResult: rootDir,
		},
		{
			name:           "Test appname",
			input:          "{{ .AppName }}",
			expectedResult: appName,
		},
		{
			name:           "Test commit",
			input:          "{{ gitCommit }}",
			expectedResult: commitID,
		},
		{
			name:  "Test {{ uuid }}",
			input: "{{ uuid }}",
			validator: func(result string) error {
				matched, err := regexp.MatchString(`(?i)^[0-9A-F]{8}-[0-9A-F]{4}-[4][0-9A-F]{3}-[89AB][0-9A-F]{3}-[0-9A-F]{12}$`, result)
				if err != nil {
					return err
				}

				if matched != true {
					return fmt.Errorf("string didnt match")
				}

				return nil
			},
		},
	}

	subject := NewGoTemplate(appName, rootDir, func() (string, error) {
		return commitID, nil
	})

	for _, tc := range testCases {
		t.Run(tc.name, func(tt *testing.T) {
			result, err := subject.Resolve(tc.input)
			assert.NoError(t, err)

			if tc.expectedResult != "" {
				assert.Equal(tt, tc.expectedResult, result)
			}

			if tc.validator != nil {
				assert.NoError(tt, tc.validator(result))
			}
		})
	}
}
