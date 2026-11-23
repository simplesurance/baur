package command

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime/pprof"
	"testing"

	"github.com/fatih/color"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/simplesurance/baur/v5/internal/exec"
	"github.com/simplesurance/baur/v5/internal/log"
)

func TestInitSb(t *testing.T) {
	initTest(t)

	// the initSb resets are done via setGlobal, to not leak settings into other tests
	resetFlags := func(t *testing.T) {
		t.Helper()

		setGlobal(t, &verboseFlag, false)
		setGlobal(t, &noColorFlag, false)
		setGlobal(t, &cpuProfilingFlag, false)
		setGlobal(t, &repositoryPath, "")
		setGlobal(t, &color.NoColor, color.NoColor)
		setGlobal(t, &exec.DefaultLogFn, exec.DefaultLogFn)
		t.Cleanup(func() { log.StdLogger.EnableDebug(false) })
	}

	t.Run("no flags changes nothing", func(t *testing.T) {
		resetFlags(t)
		wd, err := os.Getwd()
		require.NoError(t, err)
		oldColor := color.NoColor

		initSb(nil, nil)

		assert.Equal(t, oldColor, color.NoColor)
		cur, err := os.Getwd()
		require.NoError(t, err)
		assert.Equal(t, wd, cur)
	})

	t.Run("--no-color disables colors", func(t *testing.T) {
		resetFlags(t)
		color.NoColor = false
		noColorFlag = true

		initSb(nil, nil)

		assert.True(t, color.NoColor)
	})

	t.Run("--verbose enables debug logging for baur and exec", func(t *testing.T) {
		resetFlags(t)
		verboseFlag = true
		log.StdLogger.EnableDebug(false)
		exec.DefaultLogFn = nil

		initSb(nil, nil)

		assert.True(t, log.StdLogger.DebugEnabled())
		assert.NotNil(t, exec.DefaultLogFn)
	})

	t.Run("--repository changes the working directory", func(t *testing.T) {
		resetFlags(t)
		t.Chdir(t.TempDir()) // restores the original directory on cleanup
		dir := t.TempDir()
		repositoryPath = dir

		initSb(nil, nil)

		cur, err := os.Getwd()
		require.NoError(t, err)
		wantDir, err := filepath.EvalSymlinks(dir)
		require.NoError(t, err)
		gotDir, err := filepath.EvalSymlinks(cur)
		require.NoError(t, err)
		assert.Equal(t, wantDir, gotDir)
	})

	t.Run("--repository with a missing directory terminates the program", func(t *testing.T) {
		resetFlags(t)
		_, stderrBuf := interceptCmdOutput(t)
		repositoryPath = filepath.Join(t.TempDir(), "missing")

		expectExit(t, exitCodeError, func() { initSb(nil, nil) })

		assert.NotEmpty(t, stderrBuf.String())
	})

	t.Run("--cpu-prof writes a profile file", func(t *testing.T) {
		resetFlags(t)
		profFile := filepath.Join(t.TempDir(), "cpu.prof")
		setGlobal(t, &defCPUProfFile, profFile)
		cpuProfilingFlag = true
		t.Cleanup(pprof.StopCPUProfile)

		initSb(nil, nil)

		assert.FileExists(t, profFile)
	})

	t.Run("--cpu-prof with an unwritable file terminates the program", func(t *testing.T) {
		resetFlags(t)
		_, stderrBuf := interceptCmdOutput(t)
		setGlobal(t, &defCPUProfFile, filepath.Join(t.TempDir(), "missing-dir", "cpu.prof"))
		cpuProfilingFlag = true

		expectExit(t, exitCodeError, func() { initSb(nil, nil) })

		assert.NotEmpty(t, stderrBuf.String())
	})
}

// TestExecuteShowsHelp calls Execute() once, it can only be called once per
// test binary because it registers the persistent flags on the global rootCmd.
// With "go test -count=N" the repetitions after the first are skipped.
func TestExecuteShowsHelp(t *testing.T) {
	if rootCmd.PersistentFlags().Lookup("verbose") != nil {
		t.Skip("Execute() was already called in this process, it can not be called twice")
	}

	initTest(t)

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetArgs([]string{"--help"})
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetArgs(nil)
	})

	assert.NotPanics(t, Execute)

	assert.Contains(t, out.String(), "baur is a task runner for mono repositories.")
	for _, flagName := range []string{"--verbose", "--no-color", "--repository", "--cpu-prof"} {
		assert.Contains(t, out.String(), flagName)
	}

	for _, sub := range []string{"run", "status", "init", "cleanup", "release", "diff"} {
		assert.Contains(t, out.String(), sub)
	}
}
