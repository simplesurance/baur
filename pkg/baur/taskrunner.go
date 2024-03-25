package baur

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"github.com/fatih/color"

	"github.com/simplesurance/baur/v3/internal/exec"
)

type ErrUntrackedGitFilesExist struct {
	UntrackedFiles []string
}

func (e *ErrUntrackedGitFilesExist) Error() string {
	return "untracked or modified files exist in the git repository"
}

// ErrTaskRunSkipped is returned when a task run was skipped instead of executed.
var ErrTaskRunSkipped = errors.New("task run skipped")

// TaskRunner executes the command of a task.
type TaskRunner struct {
	skipEnabled         uint32 // must be accessed via atomic operations
	LogFn               exec.PrintfFn
	GitUntrackedFilesFn func(dir string) ([]string, error)
}

func NewTaskRunner() *TaskRunner {
	return &TaskRunner{
		LogFn: exec.DefaultLogFn,
	}
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
	if t.GitUntrackedFilesFn != nil {
		untracked, err := t.GitUntrackedFilesFn(task.RepositoryRoot)
		if err != nil {
			return nil, err
		}

		if len(untracked) != 0 {
			return nil, &ErrUntrackedGitFilesExist{UntrackedFiles: untracked}
		}
	}

	startTime := time.Now()
	execResult, err := exec.Command(task.Command[0], task.Command[1:]...).
		Directory(task.Directory).
		LogPrefix(color.YellowString(fmt.Sprintf("%s: ", task))).
		LogFn(t.LogFn).
		Env(append(os.Environ(), task.EnvironmentVariables...)).
		Run(context.TODO())
	if err != nil {
		return nil, err
	}

	return &RunResult{
		Result:    execResult,
		StartTime: startTime,
		StopTime:  time.Now(),
	}, nil
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
