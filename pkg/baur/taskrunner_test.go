package baur

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunningTaskFailsWhenGitWorktreeIsDirty(t *testing.T) {
	tr := NewTaskRunner()
	tr.GitUntrackedFilesFn = func(_ string) ([]string, error) {
		return []string{"1"}, nil
	}
	_, err := tr.Run(&Task{})
	var eu *ErrUntrackedGitFilesExist
	require.ErrorAs(t, err, &eu)
}

func TestEnvVarIsSet(t *testing.T) {
	tr := NewTaskRunner()
	res, err := tr.Run(&Task{
		Command:              []string{"sh", "-c", `env; if [ "$EV" = "VAL UE" ] && [ "$NOT_EXIST_EV" = "" ]; then exit 0; else exit 1; fi`},
		EnvironmentVariables: []string{"EV=VAL UE"},
	})
	require.NoError(t, err)
	require.NoError(t, res.ExpectSuccess())

}
