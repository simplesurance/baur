package baur

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v5/pkg/cfg"
)

// newTestApp writes an app config for name to <repoDir>/<relDir>/.app.toml and
// returns the App loaded from it.
func newTestApp(t *testing.T, repoDir, relDir, name string) *App {
	t.Helper()

	cfgPath := filepath.Join(repoDir, relDir, AppCfgFile)
	require.NoError(t, os.MkdirAll(filepath.Dir(cfgPath), 0o755))
	require.NoError(t, cfg.ExampleApp(name).ToFile(cfgPath))

	appCfg, err := cfg.AppFromFile(cfgPath)
	require.NoError(t, err)

	app, err := NewApp(appCfg, repoDir)
	require.NoError(t, err)

	return app
}

func TestNewApp(t *testing.T) {
	repoDir := t.TempDir()
	app := newTestApp(t, repoDir, filepath.Join("services", "api"), "api")

	assert.Equal(t, "api", app.Name)
	assert.Equal(t, "api", app.String())
	assert.Equal(t, filepath.Join("services", "api"), app.RelPath)
	assert.Equal(t, filepath.Join(repoDir, "services", "api"), app.Path)
}

func TestSortAppsByName(t *testing.T) {
	repoDir := t.TempDir()
	apps := []*App{
		newTestApp(t, repoDir, "c", "charlie"),
		newTestApp(t, repoDir, "a", "alpha"),
		newTestApp(t, repoDir, "b", "bravo"),
	}

	SortAppsByName(apps)

	names := make([]string, 0, len(apps))
	for _, a := range apps {
		names = append(names, a.Name)
	}

	assert.Equal(t, []string{"alpha", "bravo", "charlie"}, names)
}

func TestDedupApps(t *testing.T) {
	repoDir := t.TempDir()
	a1 := newTestApp(t, repoDir, "a", "alpha")
	b1 := newTestApp(t, repoDir, "b", "bravo")

	t.Run("same app listed twice is returned once", func(t *testing.T) {
		got, err := dedupApps([]*App{a1, b1, a1})
		require.NoError(t, err)

		assert.ElementsMatch(t, []*App{a1, b1}, got)
	})

	t.Run("empty input", func(t *testing.T) {
		got, err := dedupApps(nil)
		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("same name in different config files is an error", func(t *testing.T) {
		a2 := newTestApp(t, repoDir, "other", "alpha")

		_, err := dedupApps([]*App{a1, a2})

		var dupErr *ErrDuplicateAppNames
		require.ErrorAs(t, err, &dupErr)
		assert.Equal(t, "alpha", dupErr.AppName)
		assert.ElementsMatch(t,
			[]string{a1.cfg.FilePath(), a2.cfg.FilePath()},
			[]string{dupErr.AppPath1, dupErr.AppPath2},
		)
		assert.Contains(t, err.Error(), `"alpha"`)
		assert.Contains(t, err.Error(), a1.cfg.FilePath())
		assert.Contains(t, err.Error(), a2.cfg.FilePath())
	})
}

func TestDedupTasks(t *testing.T) {
	t1 := &Task{ID: "a.build"}
	t2 := &Task{ID: "a.test"}

	got := dedupTasks([]*Task{t1, t2, t1})

	assert.ElementsMatch(t, []*Task{t1, t2}, got)
	assert.Empty(t, dedupTasks(nil))
}

func TestDedupTasksSameIDKeepsLast(t *testing.T) {
	first := &Task{ID: "a.build", Name: "first"}
	last := &Task{ID: "a.build", Name: "last"}

	got := dedupTasks([]*Task{first, last})

	require.Len(t, got, 1)
	assert.Same(t, last, got[0])
}

func TestSortTasksByID(t *testing.T) {
	tasks := []*Task{{ID: "b.test"}, {ID: "a.test"}, {ID: "a.build"}}

	SortTasksByID(tasks)

	ids := make([]string, 0, len(tasks))
	for _, tk := range tasks {
		ids = append(ids, tk.ID)
	}

	assert.Equal(t, []string{"a.build", "a.test", "b.test"}, ids)
}
