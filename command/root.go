package command

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/pprof"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur/log"
	"github.com/simplesurance/baur/version"
)

var rootCmd = &cobra.Command{
	Use:              "baur",
	Short:            "baur manages builds and artifacts in mono repositories.",
	Version:          version.CurSemVer.String(),
	PersistentPreRun: initSb,
}

var verboseFlag bool
var cpuProfilingFlag bool

var defCPUProfFile = filepath.Join(os.TempDir(), "baur-cpu.prof")

func initSb(_ *cobra.Command, _ []string) {
	log.DebugEnabled = verboseFlag

	if cpuProfilingFlag {
		cpuProfFile, err := os.Create(defCPUProfFile)
		if err != nil {
			log.Fatalln(err)
		}

		if err := pprof.StartCPUProfile(cpuProfFile); err != nil {
			log.Fatalln(err)
		}
	}
}

// Execute parses commandline flags and execute their actions
func Execute() {
	rootCmd.PersistentFlags().BoolVarP(&verboseFlag, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVar(&cpuProfilingFlag, "cpu-prof", false,
		fmt.Sprintf("enable cpu profiling, result is written to %q", defCPUProfFile))

	if err := rootCmd.Execute(); err != nil {
		log.Fatalln(err)
	}

	if cpuProfilingFlag {
		log.Infof("\ncpu profile written to %q\n", defCPUProfFile)
		pprof.StopCPUProfile()
	}
}
