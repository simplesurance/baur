package exec

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCombinedStderrOutput(t *testing.T) {
	ctx := t.Context()
	const echoStr = "hello world!"

	res, err := Command("sh", "-c", fmt.Sprintf("echo -n '%s' 1>&2", echoStr)).RunCombinedOut(ctx)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, 0, res.ExitCode, "unexpected exit code")
	assert.Equal(t, echoStr, res.StrOutput(), "unexpected output")
}

func TestStdoutAndStderrIsRecordedOnErr(t *testing.T) {
	ctx := t.Context()
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
	ctx := t.Context()

	res, err := Command(
		"dd", "if=/dev/urandom", "bs=1024", fmt.Sprintf("count=%d", outBytes/1024),
	).Run(ctx)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.NoError(t, res.ExpectSuccess())

	assert.GreaterOrEqual(t, len(res.stdout.Bytes()), maxErrOutputBytesPerStream)

	// expected size: defMaxErrOutputBytesPerStream for the prefix output + defMaxErrOutputBytesPerStream for the suffix output + some bytes for the information that has been truncated
	assert.LessOrEqual(t, len(res.stdout.Bytes()), 2*maxErrOutputBytesPerStream+1024)
}

func TestCancellingRuningCommand(t *testing.T) {
	ctx, cancelFn := context.WithTimeout(t.Context(), time.Second)
	t.Cleanup(cancelFn)

	_, err := Command("sleep", "5m").Run(ctx)
	assert.Error(t, err, "command execution did not fail") //nolint:testifylint
	require.Error(t, ctx.Err(), "context err is nil")
	assert.ErrorIs(t, err, ctx.Err())
}

func TestExecDoesNotFailIfLongLinesAreStreamed(t *testing.T) {
	_, err := Command("bash", "-c",
		fmt.Sprintf("tr -d '\n' </dev/urandom | head -c %d",
			2*outputStreamLineReaderBufSiz)).LogFn(t.Logf).
		ExpectSuccess().Run(t.Context())
	require.NoError(t, err)
}

func TestStdoutStderrStream(t *testing.T) {
	ctx := t.Context()
	buf := bytes.Buffer{}
	mu := sync.Mutex{}

	_, err := Command("bash", "-c", "echo 'stdoutHello'; echo 'stderrHello' >&2; echo -n 'stdoutNoNewLine'; echo -e '\tstdoutEnd'").
		LogPrefix("").
		LogFn(func(f string, a ...any) {
			mu.Lock()
			defer mu.Unlock()
			fmt.Fprintf(&buf, f, a...)
		}).ExpectSuccess().Run(ctx)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "stdoutHello\n")
	assert.Contains(t, buf.String(), "stderrHello\n")
	assert.Contains(t, buf.String(), "stdoutNoNewLine\tstdoutEnd\n")
}

func TestStdoutStderrPrefixSaver(t *testing.T) {
	ctx := t.Context()

	// The condition is racy, run the test multiple time to make it it likely to trigger the bug.
	for i := 0; i < 100; i++ {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			result, err := Command("bash", "-c", "echo 'stdoutHello'; echo 'stderrHello' >&2; echo -n 'stdoutNoNewLine'; echo -e '\tstdoutEnd'").
				LogPrefix("").
				Run(ctx)
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, "stdoutHello\nstdoutNoNewLine\tstdoutEnd\n", string(result.stdout.Bytes()))
			assert.Equal(t, "stderrHello\n", string(result.stderr.Bytes()))
		})
	}
}
