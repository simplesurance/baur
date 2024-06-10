//go:build unix

package baur

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v4/internal/exec"
	"github.com/simplesurance/baur/v4/internal/log"
	"github.com/simplesurance/baur/v4/internal/testutils/gittest"
)

func TestSymlinkTargetFilePermissionsChange(t *testing.T) {
	for _, tc := range newGitFileTcVariations() {
		t.Run(fmt.Sprintf("commitbeforechange:%v,commitafterchange:%v",
			tc.AddToGitBeforeChange, tc.AddToGitAfterChange),
			func(t *testing.T) {
				exec.DefaultLogFn = t.Logf
				log.RedirectToTestingLog(t)
				info := prepareSymlinkTestDir(t, tc.AddToGitBeforeChange)
				require.NoError(t, os.Chmod(info.SymlinkTargetFilePath, 0755))
				if tc.AddToGitAfterChange {
					gittest.CommitFilesToGit(t, info.TempDir)
				}
				_, digestAfter := resolveInputs(t, info.Task, !tc.AddToGitAfterChange)
				require.NotEqual(t, info.TotalInputDigest.String(), digestAfter.String())
			})
	}
}
