package command

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur/v3/internal/command/term"
	"github.com/simplesurance/baur/v3/internal/exec"
	"github.com/simplesurance/baur/v3/internal/log"
	"github.com/simplesurance/baur/v3/internal/routines"
	"github.com/simplesurance/baur/v3/internal/upload/docker"
	"github.com/simplesurance/baur/v3/internal/upload/filecopy"
	"github.com/simplesurance/baur/v3/internal/upload/s3"
	"github.com/simplesurance/baur/v3/internal/vcs/git"
	"github.com/simplesurance/baur/v3/pkg/baur"
	"github.com/simplesurance/baur/v3/pkg/storage"
)

const runExample = `
baur run auth				run all tasks of the auth application, upload the produced outputs
baur run calc.check			run the check task of the calc application and upload the produced outputs
baur run *.build			run all tasks named build of the all applications and upload the produced outputs
baur run --force			run and upload all tasks of applications, independent of their status
`

const flagNameRequireCleanGitWorktree = "require-clean-git-worktree"

var runLongHelp = fmt.Sprintf(`
Execute tasks of applications.

If no argument is specified all tasks of all applications with status %s are run.

Arguments:
%s

The following Environment Variables are supported:
    %s

  S3 Upload:
    %s
    %s
    %s

  Docker Registry Upload:
    %s
    %s
    %s
    %s
`,
	term.ColoredTaskStatus(baur.TaskStatusExecutionPending),
	targetHelp,

	term.Highlight(envVarPSQLURL),

	term.Highlight("AWS_REGION"),
	term.Highlight("AWS_ACCESS_KEY_ID"),
	term.Highlight("AWS_SECRET_ACCESS_KEY"),

	term.Highlight("DOCKER_HOST"),
	term.Highlight("DOCKER_API_VERSION"),
	term.Highlight("DOCKER_CERT_PATH"),
	term.Highlight("DOCKER_TLS_VERIFY"))

var (
	statusStrSuccess = term.GreenHighlight("successful")
	statusStrSkipped = term.YellowHighlight("skipped")
	statusStrFailed  = term.RedHighlight("failed")
)

func init() {
	rootCmd.AddCommand(&newRunCmd().Command)
}

type runCmd struct {
	cobra.Command

	// Cmdline parameters
	skipUpload              bool
	force                   bool
	inputStr                []string
	lookupInputStr          string
	taskRunnerGoRoutines    uint
	showOutput              bool
	requireCleanGitWorktree bool

	// other fields
	storage      storage.Storer
	repoRootPath string
	dockerClient *docker.Client
	uploader     *baur.Uploader
	gitRepo      *git.Repository

	uploadRoutinePool     *routines.Pool
	taskRunnerRoutinePool *routines.Pool
	taskRunner            *baur.TaskRunner

	skipAllScheduledTaskRunsOnce sync.Once
	errorHappened                bool
}

type pendingTask struct {
	task   *baur.Task
	inputs *baur.Inputs
}

func newRunCmd() *runCmd {
	cmd := runCmd{
		Command: cobra.Command{
			Use:               "run [TARGET|APP_DIR]...",
			Short:             "run tasks",
			Long:              strings.TrimSpace(runLongHelp),
			Example:           strings.TrimSpace(runExample),
			ValidArgsFunction: newCompleteTargetFunc(completeTargetFuncOpts{}),
		},
	}

	cmd.Run = cmd.run

	cmd.Flags().BoolVarP(&cmd.skipUpload, "skip-upload", "s", false,
		"skip uploading task outputs and recording the run")
	cmd.Flags().BoolVarP(&cmd.force, "force", "f", false,
		"enforce running tasks independent of their status")
	cmd.Flags().StringArrayVar(&cmd.inputStr, "input-str", nil,
		"include a string as input, can be specified multiple times")
	cmd.Flags().StringVar(&cmd.lookupInputStr, "lookup-input-str", "",
		"if a run can not be found, try to find a run with this value as input-string")
	cmd.Flags().UintVarP(&cmd.taskRunnerGoRoutines, "parallel-runs", "p", 1,
		"specifies the max. number of tasks to run in parallel")
	cmd.Flags().BoolVarP(&cmd.showOutput, "show-task-output", "o", false,
		"show the output of tasks, if disabled the output is only shown "+
			"when task execution fails",
	)
	cmd.Flags().BoolVarP(&cmd.requireCleanGitWorktree, flagNameRequireCleanGitWorktree, "c", false,
		"fail if the git repository contains modified or untracked files")

	return &cmd
}

func (c *runCmd) run(_ *cobra.Command, args []string) {
	var err error

	if c.taskRunnerGoRoutines == 0 {
		stderr.Printf("--parallel-runs must be greater than 0\n")
		exitFunc(exitCodeError)
	}

	startTime := time.Now()

	repo := mustFindRepository()
	c.repoRootPath = repo.Path

	c.gitRepo = mustGetRepoState(repo.Path)

	mustUntrackedFilesNotExist(c.requireCleanGitWorktree, c.gitRepo)

	c.storage = mustNewCompatibleStorageRepo(repo)
	defer c.storage.Close()

	inputResolver := baur.NewInputResolver(
		mustGetRepoState(c.repoRootPath),
		c.repoRootPath,
		baur.AsInputStrings(c.inputStr...),
		!c.requireCleanGitWorktree,
	)
	taskStatusEvaluator := baur.NewTaskStatusEvaluator(
		c.repoRootPath,
		c.storage,
		inputResolver,
		c.lookupInputStr,
	)

	if !c.skipUpload {
		c.uploadRoutinePool = routines.NewPool(1) // run 1 upload in parallel with builds
	}

	c.taskRunnerRoutinePool = routines.NewPool(c.taskRunnerGoRoutines)
	c.taskRunner = baur.NewTaskRunner(
		baur.NewTaskInfoCreator(c.storage, taskStatusEvaluator),
	)

	if c.showOutput && !verboseFlag {
		c.taskRunner.LogFn = stderr.Printf
	}

	if c.requireCleanGitWorktree {
		c.taskRunner.GitUntrackedFilesFn = git.UntrackedFiles
	}

	c.dockerClient, err = docker.NewClient(log.StdLogger.Debugf)
	exitOnErr(err)

	s3Client, err := s3.NewClient(ctx, log.StdLogger)
	exitOnErr(err)
	c.uploader = baur.NewUploader(c.dockerClient, s3Client, filecopy.New(log.Debugf))

	if c.skipUpload {
		stdout.Printf("--skip-upload was passed, outputs won't be uploaded and task runs not recorded\n\n")
	}

	loader, err := baur.NewLoader(repo.Cfg, c.gitRepo.CommitID, log.StdLogger)
	exitOnErr(err)

	tasks, err := loader.LoadTasks(args...)
	exitOnErr(err)

	baur.SortTasksByID(tasks)

	// TODO: move taskStatusEvaluator to the cmd struct?
	pendingTasks, err := c.filterPendingTasks(taskStatusEvaluator, tasks)
	exitOnErr(err)

	if c.requireCleanGitWorktree && len(pendingTasks) == 1 {
		// if we only execute 1 task, the initial worktree check is
		// sufficient, no other tasks ran in between that could have
		// modified the worktree
		c.taskRunner.GitUntrackedFilesFn = nil
	}

	stdout.PrintSep()

	if c.force {
		stdout.Printf("Running %d/%d task(s) with status %s, %s\n\n",
			len(pendingTasks), len(tasks), term.ColoredTaskStatus(baur.TaskStatusExecutionPending), term.ColoredTaskStatus(baur.TaskStatusRunExist))
	} else {
		stdout.Printf("Running %d/%d task(s) with status %s\n\n",
			len(pendingTasks), len(tasks), term.ColoredTaskStatus(baur.TaskStatusExecutionPending))
	}

	for _, pt := range pendingTasks {
		// copy the iteration variable, to prevent that its value
		// changes in the closure to 't' of the next iteration before
		// the closure is executed
		pendingTaskCopy := pt

		c.taskRunnerRoutinePool.Queue(func() {
			task := pendingTaskCopy.task
			runResult, err := c.runTask(task)
			if err != nil {
				// error is printed in runTask()
				c.skipAllScheduledTaskRuns()
				return
			}

			outputs, err := baur.OutputsFromTask(c.dockerClient, task)
			if err != nil {
				stderr.ErrPrintln(err, task.ID)
				c.skipAllScheduledTaskRuns()
				return
			}

			if !declaredOutputsExist(task, outputs) {
				// error is printed in declaredOutputsExist()
				c.skipAllScheduledTaskRuns()
				return
			}

			if c.skipUpload {
				return
			}

			c.uploadRoutinePool.Queue(func() {
				err := c.uploadAndRecord(ctx, pendingTaskCopy, outputs, runResult)
				if err != nil {
					// error is printed in uploadAndRecord()
					c.skipAllScheduledTaskRuns()
				}
			})
		})
	}

	c.taskRunnerRoutinePool.Wait()

	if !c.skipUpload {
		stdout.Println("task execution finished, waiting for uploads to finish...")
		c.uploadRoutinePool.Wait()
	}

	stdout.PrintSep()
	stdout.Printf("finished in: %s\n",
		term.FormatDuration(
			time.Since(startTime),
		),
	)

	if c.errorHappened {
		exitFunc(exitCodeError)
	}
}

func (c *runCmd) skipAllScheduledTaskRuns() {
	c.skipAllScheduledTaskRunsOnce.Do(func() {
		c.taskRunner.SkipRuns(true)

		c.errorHappened = true

		stderr.Printf("%s, %s execution of queued task runs\n",
			term.RedHighlight("terminating"),
			term.YellowHighlight("skipping"),
		)
	})
}

func (c *runCmd) runTask(task *baur.Task) (*baur.RunResult, error) {
	result, err := c.taskRunner.Run(task)
	if err == nil {
		err = result.ExpectSuccess()
	}

	if err == nil {
		stdout.TaskPrintf(task, "execution %s (%s)\n",
			statusStrSuccess,
			term.FormatDuration(
				result.StopTime.Sub(result.StartTime),
			),
		)

		return result, nil
	}

	if errors.Is(err, baur.ErrTaskRunSkipped) {
		stderr.Printf("%s: execution %s\n",
			term.Highlight(task),
			statusStrSkipped,
		)
		return nil, err
	}

	var ee *exec.ExitCodeError
	if errors.As(err, &ee) {
		stderr.Printf("%s: %s\n",
			term.Highlight(task),
			ee.ColoredError(term.Highlight, term.RedHighlight, !c.showOutput && !verboseFlag),
		)
		return nil, err
	}

	var eUntracked *baur.ErrUntrackedGitFilesExist
	if errors.As(err, &eUntracked) {
		stderr.Println(untrackedFilesExistErrMsg(eUntracked.UntrackedFiles))
		return nil, err
	}

	stderr.Printf("%s: executing command %s: %s\n",
		term.Highlight(task),
		statusStrFailed,
		err,
	)

	return nil, err
}

func (c *runCmd) uploadAndRecord(
	ctx context.Context,
	pt *pendingTask,
	outputs []baur.Output,
	runResult *baur.RunResult,
) error {
	var uploadResults []*baur.UploadResult
	task := pt.task
	inputs := pt.inputs

	for _, output := range outputs {
		err := c.uploader.Upload(
			output,
			func(_ baur.Output, info baur.UploadInfo) {
				log.Debugf("%s: uploading output %s to %s\n",
					task, output, info)
			},
			func(o baur.Output, result *baur.UploadResult) {
				size, err := o.SizeBytes()
				if err != nil {
					stderr.ErrPrintf(err, "%s: %s", task, output)
					c.skipAllScheduledTaskRuns()
					return
				}

				bps := uint64(math.Round(float64(size) / result.Stop.Sub(result.Start).Seconds()))

				stdout.TaskPrintf(task, "%s uploaded to %s (%s/s)\n",
					output, result.URL,
					term.FormatSize(bps),
				)

				uploadResults = append(uploadResults, result)
			},
		)
		if err != nil {
			stderr.Printf("%s: %s: upload %s, %s\n",
				term.Highlight(task),
				output,
				statusStrFailed,
				err,
			)
			return err
		}
	}

	id, err := baur.StoreRun(ctx, c.storage, c.gitRepo, task, inputs, runResult, uploadResults)
	if err != nil {
		stderr.Printf("%s: recording build result in database %s, %s\n",
			term.Highlight(task),
			statusStrFailed,
			err,
		)
		return err
	}

	stdout.TaskPrintf(task, "run stored in database with ID %s\n", term.Highlight(id))

	return nil
}

func declaredOutputsExist(task *baur.Task, outputs []baur.Output) bool {
	allExist := true

	if len(outputs) == 0 {
		stdout.TaskPrintf(task, "does not produce outputs\n")
		return true
	}

	for _, output := range outputs {
		exists, err := output.Exists()
		if err != nil {
			stderr.ErrPrintf(err, task.ID)
			return false
		}

		if exists {
			size, err := output.SizeBytes()
			if err != nil {
				stderr.ErrPrintln(err, task.ID)
				return false
			}

			stdout.TaskPrintf(task, "created %s (size: %s)\n",
				output, term.FormatSize(size))

			continue
		}

		allExist = false
		stderr.TaskPrintf(task, "has %s as output defined but it was not created by the task run\n", output)
	}

	return allExist
}

func maxTaskIDLen(tasks []*baur.Task) int {
	var maxLen int

	for _, task := range tasks {
		taskIDlen := len(task.ID)

		if taskIDlen > maxLen {
			maxLen = taskIDlen
		}
	}

	return maxLen
}

func (c *runCmd) filterPendingTasks(taskStatusEvaluator *baur.TaskStatusEvaluator, tasks []*baur.Task) ([]*pendingTask, error) {
	const sep = " => "

	taskIDColLen := maxTaskIDLen(tasks) + len(sep)

	stdout.Printf("Evaluating status of tasks:\n\n")

	result := make([]*pendingTask, 0, len(tasks))
	for _, task := range tasks {
		status, inputs, run, err := taskStatusEvaluator.Status(ctx, task)
		if err != nil {
			return nil, fmt.Errorf("%s: evaluating task status failed: %w", task, err)
		}

		if status == baur.TaskStatusRunExist {
			stdout.Printf("%-*s%s%s (%s)\n",
				taskIDColLen, task, sep, term.ColoredTaskStatus(status), term.GreenHighlight(run.ID))

			if !c.force {
				continue
			}
		} else {
			stdout.Printf("%-*s%s%s\n", taskIDColLen, task, sep, term.ColoredTaskStatus(status))
		}

		result = append(result, &pendingTask{task: task, inputs: inputs})
	}

	return result, nil
}
