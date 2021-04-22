package cfg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppsWithoutAnyTasksAreValid(t *testing.T) {
	app := App{Name: "testapp"}

	err := app.Validate()
	assert.NoError(t, err)
}

func TestTaskNameValidation(t *testing.T) {
	testcases := []struct {
		TaskName       string
		ExpectedErrStr string
	}{
		{
			TaskName:       "sh.o.p",
			ExpectedErrStr: "character not allowed",
		},
		{
			TaskName:       "sh##",
			ExpectedErrStr: "character not allowed",
		},
		{
			TaskName:       "star***",
			ExpectedErrStr: "character not allowed",
		},
		{
			TaskName:       "",
			ExpectedErrStr: "can not be empty",
		},
	}
	for _, tc := range testcases {
		t.Run(tc.TaskName, func(t *testing.T) {
			a := ExampleApp("shop")
			a.Tasks[0].Name = tc.TaskName
			err := a.Validate()
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.ExpectedErrStr)
		})
	}
}
