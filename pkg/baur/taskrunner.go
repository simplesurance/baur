package baur

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"github.com/fatih/color"

	"github.com/simplesurance/baur/v5/internal/exec"
)

type ErrUntrackedGitFilesExist struct {
	UntrackedFiles []string
}

func (e *ErrUntrackedGitFilesExist) Error() string {
	return "untracked or modified files exist in the git repository"
}

// ErrTaskRunSkipped is returned when a task run was skipped instead of executed.
var ErrTaskRunSkipped = errors.New("task run skipped")

type TaskInfoRetriever interface {
	Inputs(*Task) (*Inputs, error)
	Task(id string) (*Task, error)
}

// TaskRunner executes the command of a task.
type TaskRunner struct {
	skipEnabled         uint32 // must be accessed via atomic operations
	LogFn               exec.PrintfFn
	GitUntrackedFilesFn func(dir string) ([]string, error)
	taskInfoCreator     *TaskInfoCreator
}

func NewTaskRunner(taskInfoCreator *TaskInfoCreator) *TaskRunner {
	return &TaskRunner{
		LogFn:           exec.DefaultLogFn,
		taskInfoCreator: taskInfoCreator,
	}
}

// RunResult represents the results of a task run.
type RunResult struct {
	*exec.Result
	StartTime time.Time
	StopTime  time.Time
}

func (t *TaskRunner) deleteTmpFiles(paths []string) {
	for _, path := range paths {
		// TODO: install a signal handler that deletes the
		// file on termination. This ensures the temp. files are deleted if baur
		// gets terminated.
		if err := os.Remove(path); err != nil {
			// TODO: always log this, not only with debug priority
			t.LogFn("deleting temporary task info file %q failed: %s\n", path, err)
		}
	}

}

func (t *TaskRunner) createTaskInfoEnv(ctx context.Context, task *Task) ([]string, func(), error) {
	env := make([]string, 0, len(task.TaskInfoDependencies))
	tmpfilepaths := make([]string, 0, len(task.TaskInfoDependencies))

	for _, ti := range task.TaskInfoDependencies {
		content, err := t.taskInfoCreator.CreateFileContent(ctx, ti.Task)
		if err != nil {
			t.deleteTmpFiles(tmpfilepaths)
			return nil, nil, fmt.Errorf("generating TaskInfo content of %q failed: %w", task.ID, err)
		}

		path, err := content.ToTmpfile(ti.Task.ID)
		if err != nil {
			t.deleteTmpFiles(tmpfilepaths)
			return nil, nil, fmt.Errorf("writing task info for %q to file failed: %w", ti.Task.ID, err)
		}

		tmpfilepaths = append(tmpfilepaths, path)
		env = append(env, ti.EnvVarName+"="+path)
	}

	return env, func() { t.deleteTmpFiles(tmpfilepaths) }, nil
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

	env, deleteTempTaskInfoFilesFn, err := t.createTaskInfoEnv(context.TODO(), task)
	if err != nil {
		return nil, err
	}
	defer deleteTempTaskInfoFilesFn()

	startTime := time.Now()
	execResult, err := exec.Command(task.Command[0], task.Command[1:]...).
		Directory(task.Directory).
		LogPrefix(color.YellowString(fmt.Sprintf("%s: ", task))).
		LogFn(t.LogFn).
		Env(append(os.Environ(), env...)).
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
