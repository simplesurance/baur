package exec

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCombinedStderrOutput(t *testing.T) {
	ctx := context.Background()
	const echoStr = "hello world!"

	res, err := Command("sh", "-c", fmt.Sprintf("echo -n '%s' 1>&2", echoStr)).RunCombinedOut(ctx)
	require.NoError(t, err)

	if res.ExitCode != 0 {
		t.Fatal(res.ExpectSuccess())
	}

	assert.Equal(t, echoStr, res.StrOutput(), "unexpected output")
}

func TestStdoutAndStderrIsRecordedOnErr(t *testing.T) {
	ctx := context.Background()
	const stdoutOutput = "hello stdout"
	const stderrOutput = "hello stderr"

	res, err := Command(
		"sh", "-c",
		fmt.Sprintf("echo -n '%s' 1>&2; echo -n '%s'; exit 1", stderrOutput, stdoutOutput),
	).Run(ctx)
	require.NoError(t, err)
	require.Error(t, res.ExpectSuccess())

	assert.Equal(t, stdoutOutput, string(res.stdout.Bytes()), "unexpected stdout output")
	assert.Equal(t, stderrOutput, string(res.stderr.Bytes()), "unexpected stderr output")
}

func TestLongStdoutOutputIsTruncated(t *testing.T) {
	const outBytes = 100 * 1024 * 1024
	ctx := context.Background()

	res, err := Command(
		"dd", "if=/dev/urandom", "bs=1024", fmt.Sprintf("count=%d", outBytes/1024),
	).
		Run(ctx)
	require.NoError(t, err)
	require.NoError(t, res.ExpectSuccess())

	assert.GreaterOrEqual(t, len(res.stdout.Bytes()), defMaxErrOutputBytesPerStream)

	// expected size: defMaxErrOutputBytesPerStream for the prefix output + defMaxErrOutputBytesPerStream for the suffix output + some bytes for the information that has been truncated
	assert.LessOrEqual(t, len(res.stdout.Bytes()), 2*defMaxErrOutputBytesPerStream+1024)
}

func TestCancellingRuningCommand(t *testing.T) {
	ctx, cancelFn := context.WithTimeout(context.Background(), time.Second)
	t.Cleanup(cancelFn)

	_, err := Command("sleep", "5m").Run(ctx)
	assert.Error(t, err, "command execution did not fail") //nolint:testifylint
	require.Error(t, ctx.Err(), "context err is nil")
	assert.ErrorIs(t, err, ctx.Err()) //nolint:testifylint
	t.Log(err)

}
