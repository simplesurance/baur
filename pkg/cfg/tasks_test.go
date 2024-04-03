package cfg

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAppsWithoutAnyTasksAreValid(t *testing.T) {
	app := App{Name: "testapp"}

	err := app.Validate()
	require.NoError(t, err)
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

func TestTaskInfosAreCycleFree_SelfRef(t *testing.T) {
	tasks := Tasks{
		{
			Name: "a",
			Input: Input{
				TaskInfos: []TaskInfo{
					{TaskName: "a"},
				},
			},
		},
	}
	require.Error(t, tasks.validateTaskInfosAreCycleFree())
}

func TestTaskInfosAreCycleFree_SimpleLoop(t *testing.T) {
	tasks := Tasks{
		{
			Name: "a",
			Input: Input{
				TaskInfos: []TaskInfo{
					{TaskName: "b"},
				},
			},
		},

		{
			Name: "b",
			Input: Input{
				TaskInfos: []TaskInfo{
					{TaskName: "a"},
				},
			},
		},
	}
	require.Error(t, tasks.validateTaskInfosAreCycleFree())
}

func TestTaskInfosAreCycleFree_DeepLoop(t *testing.T) {
	tasks := Tasks{
		{
			Name: "a",
			Input: Input{
				TaskInfos: []TaskInfo{
					{TaskName: "b"},
				},
			},
		},

		{
			Name: "b",
			Input: Input{
				TaskInfos: []TaskInfo{
					{TaskName: "c"},
				},
			},
		},
		{
			Name: "c",
			Input: Input{
				TaskInfos: []TaskInfo{
					{TaskName: "d"},
				},
			},
		},
		{
			Name: "d",
			Input: Input{
				TaskInfos: []TaskInfo{
					{TaskName: "b"},
				},
			},
		},
	}
	err := tasks.validateTaskInfosAreCycleFree()
	require.Error(t, err)
	t.Log(err)
}
