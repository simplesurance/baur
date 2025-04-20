//go:build dbtest
// +build dbtest

package command

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v5/internal/testutils/repotest"
)

// TestShowArgs verifies that the show command works with all supported
// parameters to specify the app or task
func TestShowArgs(t *testing.T) {
	r := repotest.CreateBaurRepository(t, repotest.WithNewDB())
	app := r.CreateSimpleApp(t)

	showCmd := newShowCmd()

	t.Run("appName", func(t *testing.T) {
		initTest(t)
		showCmd.Run(&showCmd.Command, []string{app.Name})
	})

	t.Run("taskName", func(t *testing.T) {
		initTest(t)
		showCmd.Run(&showCmd.Command, []string{
			fmt.Sprintf("%s.%s", app.Name, app.Tasks[0].Name),
		})
	})

	t.Run("appRelDir", func(t *testing.T) {
		initTest(t)
		appDir := filepath.Dir(app.FilePath())
		relDir, err := filepath.Rel(r.Dir, appDir)
		require.NoError(t, err)

		showCmd.Run(&showCmd.Command, []string{relDir})
	})

	t.Run("appCurrentDir", func(t *testing.T) {
		initTest(t)
		appDir := filepath.Dir(app.FilePath())

		t.Chdir(appDir)

		showCmd.Run(&showCmd.Command, []string{"."})
	})
}

func TestShowWithRepositoryArg(t *testing.T) {
	r := repotest.CreateBaurRepository(t, repotest.WithNewDB())
	app := r.CreateSimpleApp(t)

	t.Chdir(os.TempDir())
	oldRepoPath := repositoryPath
	t.Cleanup(func() {
		repositoryPath = oldRepoPath
		rootCmd.SetArgs(os.Args[1:])
	})

	rootCmd.SetArgs([]string{"show", app.Name})
	_, stderrBuf := interceptCmdOutput(t)
	require.Panics(t, func() { require.NoError(t, rootCmd.Execute()) })
	require.Contains(t, stderrBuf.String(), "baur repository not found")

	repositoryPath = r.Dir
	require.NoError(t, rootCmd.Execute())
}
