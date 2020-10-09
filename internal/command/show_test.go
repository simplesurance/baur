// +build dbtest

package command

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v1/internal/testutils/repotest"
)

// TestShowArgs verifies that the show command works with all supported
// parameters to specify the app or task
func TestShowArgs(t *testing.T) {
	initTest(t)
	r := repotest.CreateBaurRepository(t, repotest.WithNewDB())
	app := r.CreateSimpleApp(t)

	showCmd := newShowCmd()

	t.Run("appName", func(t *testing.T) {
		showCmd.Command.Run(&showCmd.Command, []string{app.Name})
	})

	t.Run("taskName", func(t *testing.T) {
		showCmd.Command.Run(&showCmd.Command, []string{
			fmt.Sprintf("%s.%s", app.Name, app.Tasks[0].Name),
		})
	})

	t.Run("appRelDir", func(t *testing.T) {
		appDir := filepath.Dir(app.FilePath())
		relDir, err := filepath.Rel(r.Dir, appDir)
		require.NoError(t, err)

		showCmd.Command.Run(&showCmd.Command, []string{relDir})
	})

	t.Run("appCurrentDir", func(t *testing.T) {
		appDir := filepath.Dir(app.FilePath())

		err := os.Chdir(appDir)
		require.NoError(t, err)

		showCmd.Command.Run(&showCmd.Command, []string{"."})
	})

}
