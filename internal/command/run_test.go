// +build dbtest

package command

import (
	"fmt"
	"testing"

	"github.com/simplesurance/baur/v2/internal/testutils/repotest"
	"github.com/simplesurance/baur/v2/pkg/cfg"
	"github.com/stretchr/testify/require"
)

func TestRunSimultaneously(t *testing.T) {
	initTest(t)

	r := repotest.CreateBaurRepository(t, repotest.WithNewDB())

	testApps := 6
	apps := make([]*cfg.App,testApps)

	for i := 0; i < testApps; i++ {
		apps[i] = r.CreateAppWithNoOutputs(t, fmt.Sprintf("myapp-%d", i))
	}

	doInitDb(t)

	_, _ = interceptCmdOutput(t)

	runCmdTest := newRunCmd()
	runCmdTest.SetArgs([]string{"-t", "3"})
	err := runCmdTest.Execute()
	require.NoError(t, err)
}
