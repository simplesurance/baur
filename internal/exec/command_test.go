package exec

import (
	"context"
	"fmt"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCombinedStdoutOutput(t *testing.T) {
	ctx := context.Background()
	const echoStr = "hello World!"

	var res *ResultOut
	var err error
	if runtime.GOOS == "windows" {
		res, err = Command("bash", "-c", fmt.Sprintf("echo -n '%s'", echoStr)).RunCombinedOut(ctx)

	} else {
		res, err = Command("echo", "-n", echoStr).RunCombinedOut(ctx)
	}

	require.NoError(t, err)
	require.Equal(t, 0, res.ExitCode, "exit code is not 0")
	assert.True(t, res.success, "result.succces is not true")
	assert.Nil(t, res.ExpectSuccess(), "expect success returns an error") //nolint:testifylint
	assert.Equal(t, echoStr, res.StrOutput())
}

func TestCommandFails(t *testing.T) {
	ctx := context.Background()
	var res *ResultOut
	var err error
	if runtime.GOOS == "windows" {
		res, err = Command("cmd", "/C", "exit", "1").RunCombinedOut(ctx)
	} else {
		res, err = Command("false").RunCombinedOut(ctx)
	}

	require.NoError(t, err)
	assert.Equal(t, 1, res.ExitCode, "unexpected exit code")
	assert.Empty(t, res.CombinedOutput, "process output not empty")
	require.Error(t, res.ExpectSuccess())
}

func TestExpectSuccess(t *testing.T) {
	ctx := context.Background()
	var res *Result
	var err error
	if runtime.GOOS == "windows" {
		res, err = Command("cmd", "/C", "exit", "1").ExpectSuccess().Run(ctx)
	} else {
		res, err = Command("false").ExpectSuccess().Run(ctx)
	}
	require.Error(t, err)
	assert.Nil(t, res)
}
