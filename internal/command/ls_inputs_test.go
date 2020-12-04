// +build dbtest

package command

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v1"
	"github.com/simplesurance/baur/v1/internal/testutils/repotest"
)

func TestLsInputsTaskAndRunInputsAreTheSame(t *testing.T) {
	initTest(t)

	r := repotest.CreateBaurRepository(t, repotest.WithNewDB())
	app := r.CreateAppWithNoOutputs(t, "myapp")
	doInitDb(t)

	taskSpec := fmt.Sprintf("%s.%s", app.Name, app.Tasks[0].Name)

	stdout, _ := interceptCmdOutput(t)

	lsInputsCmd := newLsInputsCmd()
	lsInputsCmd.SetArgs([]string{"--csv", "--digests", taskSpec})
	err := lsInputsCmd.Execute()
	require.NoError(t, err)

	lsTaskOutput := stdout.String()

	runCmd := newRunCmd()
	runCmd.run(&runCmd.Command, []string{taskSpec})

	stdout.Reset()
	lsInputsCmd.SetArgs([]string{"--csv", "--digests", "1"})
	err = lsInputsCmd.Execute()
	require.NoError(t, err)

	appInputFile := fmt.Sprintf("%s/%s.txt", app.Name, app.Name)
	appTomlFile := fmt.Sprintf("%s/%s", app.Name, baur.AppCfgFile)

	lsTaskRunOutput := stdout.String()
	assert.Contains(t, lsTaskRunOutput, appInputFile)
	assert.Contains(t, lsTaskRunOutput, appTomlFile)

	assert.Equal(t, lsTaskOutput, lsTaskRunOutput)
}
