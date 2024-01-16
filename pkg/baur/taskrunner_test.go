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
