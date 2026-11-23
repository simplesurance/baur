//go:build dbtest

package command

import (
	"encoding/csv"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v5/internal/testutils/fstest"
)

const (
	diffExitSame   = 0
	diffExitError  = 1
	diffExitDiffer = 2
)

func diffInputs(t *testing.T, expectedExitCode int, args ...string) (stdout, stderr string) {
	t.Helper()

	stdoutBuf, stderrBuf := interceptCmdOutput(t)

	cmd := newDiffInputsCmd()
	cmd.SetArgs(args)
	execCheck(t, cmd, expectedExitCode)

	return stdoutBuf.String(), stderrBuf.String()
}

func TestDiffInputsTaskRunReferences(t *testing.T) {
	r := newRepoWithTasks(t, shTask("a", "true", nil, "in.txt"))
	inFile := filepath.Join(r.Dir, testAppName, "in.txt")

	fstest.WriteToFile(t, []byte("1"), inFile)
	runTasks(t, 0) // run id 1

	fstest.WriteToFile(t, []byte("2"), inFile)
	runTasks(t, 0) // run id 2
	require.Len(t, storedTaskRunIDs(t, r), 2)

	t.Run("last run and the run before differ in the changed file", func(t *testing.T) {
		out, _ := diffInputs(t, diffExitDiffer, "--format=csv", "-q", "svc.a^^", "svc.a^")

		recs, err := csv.NewReader(strings.NewReader(out)).ReadAll()
		require.NoError(t, err)
		require.Len(t, recs, 1, out)
		assert.Equal(t, "D", recs[0][0])
		assert.Equal(t, "svc/in.txt", recs[0][1])
		assert.NotEmpty(t, recs[0][2])
		assert.NotEmpty(t, recs[0][3])
		assert.NotEqual(t, recs[0][2], recs[0][3])
	})

	t.Run("run ids can be used instead of task references", func(t *testing.T) {
		diffInputs(t, diffExitDiffer, "1", "2")
	})

	t.Run("current inputs equal to the last run", func(t *testing.T) {
		out, _ := diffInputs(t, diffExitSame, "svc.a", "svc.a^")

		assert.Contains(t, out, "the inputs are the same")
	})

	t.Run("current inputs differ from the last run", func(t *testing.T) {
		fstest.WriteToFile(t, []byte("3"), inFile)
		t.Cleanup(func() { fstest.WriteToFile(t, []byte("2"), inFile) })

		out, _ := diffInputs(t, diffExitDiffer, "svc.a", "svc.a^")

		assert.Contains(t, out, "the inputs differ")
		assert.Contains(t, out, "svc/in.txt")
	})

	t.Run("--input-str is added to the inputs of the first argument", func(t *testing.T) {
		out, _ := diffInputs(t, diffExitDiffer, "--input-str=extra", "svc.a", "svc.a^")

		assert.Contains(t, out, "string:extra")
	})

	t.Run("same run referenced twice", func(t *testing.T) {
		_, errOut := diffInputs(t, diffExitError, "1", "1")
		assert.Contains(t, errOut, "refer to the same task-run")

		_, errOut = diffInputs(t, diffExitError, "svc.a", "svc.a")
		assert.Contains(t, errOut, "refer to the same task-run")

		_, errOut = diffInputs(t, diffExitError, "svc.a^", "2")
		assert.Contains(t, errOut, "refer to the same task-run")
	})

	t.Run("run that does not exist", func(t *testing.T) {
		_, errOut := diffInputs(t, diffExitError, "svc.a^^^", "svc.a")
		assert.Contains(t, errOut, "does not exist, only 2 task-run(s) exist(s)")

		diffInputs(t, diffExitError, "9999", "1")
	})

	t.Run("task that does not exist", func(t *testing.T) {
		diffInputs(t, diffExitError, "svc.nope", "svc.a")
	})

	t.Run("invalid arguments are rejected by the command", func(t *testing.T) {
		initTest(t)

		for _, args := range [][]string{{"svc.a"}, {"svc.a", "svc.a^", "1"}, {"svc", "1"}, {"a b", "1"}} {
			cmd := newDiffInputsCmd()
			cmd.SetArgs(args)
			require.Error(t, cmd.Execute(), args)
		}
	})
}
