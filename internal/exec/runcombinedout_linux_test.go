package exec

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// RunCombinedOut collects stdout and stderr in one buffer. Both streams are
// copied by separate goroutines. The test fails with the race detector if
// writes to the buffer are not synchronized.
func TestRunCombinedOutWithConcurrentStdoutAndStderr(t *testing.T) {
	const lines = 500

	script := fmt.Sprintf(
		`i=0; while [ $i -lt %d ]; do echo "out$i"; echo "err$i" >&2; i=$((i+1)); done`,
		lines,
	)

	res, err := Command("sh", "-c", script).ExpectSuccess().RunCombinedOut(t.Context())
	require.NoError(t, err)

	got := strings.Split(strings.TrimSuffix(res.StrOutput(), "\n"), "\n")
	assert.Len(t, got, 2*lines, "no output of stdout or stderr must be lost")

	assert.Contains(t, got, "out0")
	assert.Contains(t, got, fmt.Sprintf("out%d", lines-1))
	assert.Contains(t, got, "err0")
	assert.Contains(t, got, fmt.Sprintf("err%d", lines-1))
}
