package baur

import (
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/fatih/color"

	"github.com/simplesurance/baur/v3/internal/exec"
)

// ErrTaskRunSkipped is returned when a task run was skipped instead of executed.
var ErrTaskRunSkipped = errors.New("task run skipped")

// TaskRunner executes the command of a task.
type TaskRunner struct {
	skipEnabled uint32 // must be accessed via atomic operations
}

func NewTaskRunner() *TaskRunner {
	return &TaskRunner{}
}

// RunResult represents the results of a task run.
type RunResult struct {
	*exec.Result
	StartTime time.Time
	StopTime  time.Time
}

// Run executes the command of a task and returns the execution result.
// The output of the commands are logged with debug log level.
func (t *TaskRunner) Run(task *Task) (*RunResult, error) {
	startTime := time.Now()

	if t.SkipRunsIsEnabled() {
		return nil, ErrTaskRunSkipped
	}

	// TODO: rework exec, stream the output instead of storing all in memory
	execResult, err := exec.Command(task.Command[0], task.Command[1:]...).
		Directory(task.Directory).
		DebugfPrefix(color.YellowString(fmt.Sprintf("%s: ", task))).
		Run()
	if err != nil {
		return nil, err
	}

	return &RunResult{
		Result:    execResult,
		StartTime: startTime,
		StopTime:  time.Now(),
	}, nil
}

// TODO: DUPLICATE FROM INTERNAL/COMMAND/SHELL.GO, DO NOT DO THIS
var alwaysAllowed = []string{
	".baur.toml",
	".git",
}

func (t *TaskRunner) RunIsolated(task *Task, inputs Inputs) (*RunResult, error) {
	startTime := time.Now()

	if t.SkipRunsIsEnabled() {
		return nil, ErrTaskRunSkipped
	}

	// TODO: FIX HARDCODED VERBOSE FLAG
	args := []string{
		fmt.Sprintf("--verbose=%t", false),
		"__sandbox_reexec",
	}

	reExecInfoBuf, err := (&exec.ReExecInfo{
		RepositoryDir:       task.RepositoryRoot,
		OverlayFsTmpDir:     "/home/fho/tmp", // TODO: CHANGE THIS!!!!!!!!!!!
		Command:             task.Command,
		WorkingDirectory:    task.Directory,
		AllowedFilesRelPath: append(relFileInputPaths(inputs.Inputs()), alwaysAllowed...),
	}).Encode()

	execResult, err := exec.Command("/proc/self/exe", args...).
		DebugfPrefix(color.YellowString(fmt.Sprintf("%s: ", task))).
		RunInNs(reExecInfoBuf)
	if err != nil {
		return nil, err
	}

	return &RunResult{
		Result:    execResult,
		StartTime: startTime,
		StopTime:  time.Now(),
	}, nil
}

// TODO: THIS IS DUPLICATED FROM INTERNAL/COMMAND/SHELL.GO, DO NOT DO THAT!
func relFileInputPaths(inputs []Input) []string {
	result := make([]string, 0, len(inputs))

	for _, input := range inputs {
		inputFile, ok := input.(*InputFile)
		if !ok {
			continue
		}

		result = append(result, inputFile.RelPath())
	}

	return result
}

func (t *TaskRunner) setSkipRuns(val uint32) {
	atomic.StoreUint32(&t.skipEnabled, val)
}

// SkipRuns can be enabled to skip all further executions of tasks by Run().
func (t *TaskRunner) SkipRuns(enabled bool) {
	if enabled {
		t.setSkipRuns(1)
	} else {
		t.setSkipRuns(0)
	}
}

// SkipRunsIsEnabled returns true if SkipRuns is enabled.
func (t *TaskRunner) SkipRunsIsEnabled() bool {
	return atomic.LoadUint32(&t.skipEnabled) == 1
}
