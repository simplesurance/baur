package baur

import (
	"fmt"
	"time"

	"github.com/fatih/color"

	"github.com/simplesurance/baur/v1/internal/exec"
)

type TaskRunner struct{}

func NewTaskRunner() *TaskRunner {
	return &TaskRunner{}
}

type RunResult struct {
	*exec.Result
	StartTime time.Time
	StopTime  time.Time
}

func (t *TaskRunner) Run(task *Task) (*RunResult, error) {
	startTime := time.Now()

	// TODO: rework exec, stream the output instead of storing all in memory
	execResult, err := exec.ShellCommand(task.Command).
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
