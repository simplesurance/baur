package command

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime/pprof"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/simplesurance/baur/v1/internal/command/term"
	"github.com/simplesurance/baur/v1/internal/exec"
	"github.com/simplesurance/baur/v1/internal/log"
	"github.com/simplesurance/baur/v1/internal/version"
)

var rootCmd = &cobra.Command{
	Use:              "baur",
	Short:            "baur is a task runner for mono repositories.",
	PersistentPreRun: initSb,
}

var verboseFlag bool
var cpuProfilingFlag bool
var noColorFlag bool

var defCPUProfFile = filepath.Join(os.TempDir(), "baur-cpu.prof")

var ctx = context.Background()

var stdout = term.NewStream(os.Stdout)
var stderr = term.NewStream(os.Stderr)

var exitFunc = func(code int) { os.Exit(code) }

func initSb(_ *cobra.Command, _ []string) {
	if verboseFlag {
		log.StdLogger.EnableDebug(verboseFlag)
		exec.DefaultDebugfFn = log.StdLogger.Debugf
	}

	if noColorFlag {
		color.NoColor = true
	}

	if cpuProfilingFlag {
		cpuProfFile, err := os.Create(defCPUProfFile)
		exitOnErr(err)

		err = pprof.StartCPUProfile(cpuProfFile)
		exitOnErr(err)
	}
}

// Execute parses commandline flags and execute their actions
func Execute() {
	if err := version.LoadPackageVars(); err != nil {
		stderr.Printf("setting version failed: %s\n", err)
	}
	rootCmd.Version = version.CurSemVer.String()

	rootCmd.PersistentFlags().BoolVarP(&verboseFlag, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVar(&cpuProfilingFlag, "cpu-prof", false,
		fmt.Sprintf("enable cpu profiling, result is written to %q", defCPUProfFile))
	rootCmd.PersistentFlags().BoolVar(&noColorFlag, "no-color", false, "disable color output")

	err := rootCmd.Execute()
	exitOnErr(err)

	if cpuProfilingFlag {
		stdout.Printf("\ncpu profile written to %q\n", defCPUProfFile)
		pprof.StopCPUProfile()
	}
}
