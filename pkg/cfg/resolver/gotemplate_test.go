package resolver

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
			"Test env",
			fmt.Sprintf("test {{ env \"%s\" }} {{ env \"%s\" }}bye", envVar, envVar),
			fmt.Sprintf("test %s %sbye", envVarVal, envVarVal),
			nil,
		},
		{
			"Test root",
			"{{ .Root }}",
			rootDir,
			nil,
		},
		{
			"Test appname",
			"{{ .AppName }}",
			appName,
			nil,
		},
		{
			"Test commit",
			"{{ gitCommit }}",
			commitID,
			nil,
		},
		{
			"Test {{ uuid }}",
			"{{ uuid }}",
			"",
			func(result string) error {
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
