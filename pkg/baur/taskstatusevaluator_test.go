package baur

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v3/pkg/cfg"
)

func TestStatusEvaluatorFailsWhenTaskInfoAndLookupStrIsUsed(t *testing.T) {
	te := NewTaskStatusEvaluator("", nil, nil, "blubb")

	task := Task{
		UnresolvedInputs: &cfg.Input{
			TaskInfos: []cfg.TaskInfo{{TaskName: "abc"}},
		},
	}

	_, _, _, err := te.Status(context.Background(), &task)
	require.ErrorContains(t, err, "'--lookup-input-str' is unsupported")
}

func TestReplaceInputStrings(t *testing.T) {
	inputs := []Input{
		NewInputString("before"),
	}
	result := replaceInputStrings(NewInputs(inputs), []Input{NewInputString("after")})
	require.Len(t, result.inputs, 1)

	require.Contains(t, result.inputs[0].String(), "after")
}
