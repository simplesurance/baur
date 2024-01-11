package exec

import (
	"context"
	"fmt"
	"runtime"
	"testing"

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

	if err != nil {
		t.Fatal(err)
	}

	if res.ExitCode != 0 {
		t.Fatal(res.ExpectSuccess())
	}

	if res.StrOutput() != echoStr {
		t.Errorf("expected output '%s', got '%s'", echoStr, res.StrOutput())
	}
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
	if err != nil {
		t.Fatal(err)
	}

	if res.ExitCode != 1 {
		t.Fatalf("cmd exited with code %d, expected 1", res.ExitCode)
	}

	if len(res.CombinedOutput) != 0 {
		t.Fatalf("expected no output from command but got '%s'", res.StrOutput())
	}
	require.Error(t, res.ExpectSuccess())
	t.Log(res.ExpectSuccess())
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
	if err == nil {
		t.Fatal("Command did not return an error")
	}

	if res != nil {
		t.Fatalf("Command returned an error and result was not nil: %+v", res)
	}

}
