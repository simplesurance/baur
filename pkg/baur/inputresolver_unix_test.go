//go:build unix

package baur

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v3/internal/exec"
	"github.com/simplesurance/baur/v3/internal/log"
	"github.com/simplesurance/baur/v3/internal/testutils/gittest"
)

func TestSymlinkTargetFilePermissionsChange(t *testing.T) {
	for _, tc := range newGitFileTcVariations() {
		t.Run(fmt.Sprintf("gitrepo:%+v,commitbeforechange:%v,commitafterchange:%v",
			tc.CreateGitRepo, tc.AddToGitBeforeChange, tc.AddToGitAfterChange),
			func(t *testing.T) {
				exec.DefaultLogFn = t.Logf
				log.RedirectToTestingLog(t)
				info := prepareSymlinkTestDir(t, tc.CreateGitRepo, tc.AddToGitBeforeChange)
				require.NoError(t, os.Chmod(info.SymlinkTargetFilePath, 0755))
				if tc.AddToGitAfterChange {
					gittest.CommitFilesToGit(t, info.TempDir)
				}
				digestAfter := resolveInputs(t, info.Task)
				require.NotEqual(t, info.TotalInputDigest.String(), digestAfter.String())
			})
	}
}
