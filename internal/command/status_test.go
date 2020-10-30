// +build dbtest

package command

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v1/internal/testutils/repotest"
)

func TestStatusCombininingFieldAndStatusParameters(t *testing.T) {
	initTest(t)

	r := repotest.CreateBaurRepository(t, repotest.WithNewDB())
	app := r.CreateSimpleApp(t)

	runInitDb(t)
	stdoutBuf, _ := interceptCmdOutput(t)
	statusCmd := newStatusCmd()
	statusCmd.SetArgs([]string{"-f", "task-id", "-s", "pending"})
	statusCmd.Execute()

	require.Contains(t, stdoutBuf.String(), app.Name)
}
