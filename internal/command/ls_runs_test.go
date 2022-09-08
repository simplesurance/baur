//go:build dbtest
// +build dbtest

package command

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v2/internal/testutils/repotest"
)

func TestLsRunsHasInput(t *testing.T) {
	initTest(t)

	r := repotest.CreateBaurRepository(t, repotest.WithNewDB())
	app := r.CreateSimpleApp(t)
	runInitDb(t)

	taskID := fmt.Sprintf("%s.%s", app.Name, app.Tasks[0].Name)

	runCmd := newRunCmd()
	runCmd.inputStr = []string{"hello"}
	runCmd.Command.Run(&runCmd.Command, []string{taskID})

	stdoutBuf, _ := interceptCmdOutput(t)
	lsRunsCmd := newLsRunsCmd()
	lsRunsCmd.csv = true

	relAppCfgPath, err := filepath.Rel(r.Dir, app.FilePath())
	require.NoError(t, err)

	lsRunsCmd.input = relAppCfgPath
	lsRunsCmd.Run(&lsRunsCmd.Command, []string{taskID})
	assert.Contains(t, stdoutBuf.String(), fmt.Sprintf("1,%s,%s,", app.Name, app.Tasks[0].Name))

	lsRunsCmd.input = fmt.Sprintf("string:%s", runCmd.inputStr[0])
	lsRunsCmd.Run(&lsRunsCmd.Command, []string{taskID})
	assert.Contains(t, stdoutBuf.String(), fmt.Sprintf("1,%s,%s,", app.Name, app.Tasks[0].Name))

	lsRunsCmd.input = "nonononodoesnotexist"
	var cmdFailed bool

	exitFunc = func(code int) {
		cmdFailed = code != 0
	}

	lsRunsCmd.Run(&lsRunsCmd.Command, []string{taskID})
	assert.True(t, cmdFailed)
}
