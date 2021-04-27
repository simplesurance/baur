package baur

import (
	"fmt"
	"time"

	"github.com/fatih/color"

	"github.com/simplesurance/baur/v2/internal/exec"
)

// TaskRunner executes the command of a task.
type TaskRunner struct{}

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
