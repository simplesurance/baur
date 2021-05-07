package command

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur/v2/internal/command/term"
	"github.com/simplesurance/baur/v2/internal/log"
	"github.com/simplesurance/baur/v2/internal/routines"
	"github.com/simplesurance/baur/v2/internal/upload/docker"
	"github.com/simplesurance/baur/v2/internal/upload/filecopy"
	"github.com/simplesurance/baur/v2/internal/upload/s3"
	"github.com/simplesurance/baur/v2/internal/vcs"
	"github.com/simplesurance/baur/v2/pkg/baur"
	"github.com/simplesurance/baur/v2/pkg/storage"
)

const runExample = `
baur run auth				run all tasks of the auth application, upload the produced outputs
baur run calc.check			run the check task of the calc application and upload the produced outputs
baur run *.build			run all tasks named build of the all applications and upload the produced outputs
baur run --force			run and upload all tasks of applications, independent of their status
`

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

func init() {
	rootCmd.AddCommand(&newRunCmd().Command)
}

type runCmd struct {
	cobra.Command

	// Cmdline parameters
	skipUpload     bool
	force          bool
	inputStr       []string
	lookupInputStr string
	taskRunners    uint

	// other fields
	storage      storage.Storer
	repoRootPath string
	dockerClient *docker.Client
	uploader     *baur.Uploader
	vcsState     vcs.StateFetcher

	uploadRoutinePool     *routines.Pool
	taskRunnerRoutinePool *routines.Pool
}

func newRunCmd() *runCmd {
	cmd := runCmd{
		Command: cobra.Command{
			Use:     "run [TARGET|APP_DIR]...",
			Short:   "run tasks",
			Long:    runLongHelp,
			Example: strings.TrimSpace(runExample),
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
	cmd.Flags().UintVarP(&cmd.taskRunners, "tasks", "t", 1,
		"specifies the number of tasks (builds) to run simultaneously")

	return &cmd
}

func (c *runCmd) run(cmd *cobra.Command, args []string) {
	var err error

	if c.taskRunners == 0 {
		exitOnErr(fmt.Errorf("the number of tasks must be greater than 0"))
	}

	startTime := time.Now()

	repo := mustFindRepository()
	c.repoRootPath = repo.Path

	c.storage = mustNewCompatibleStorage(repo)
	defer c.storage.Close()

	c.uploadRoutinePool = routines.NewPool(1) // run 1 upload in parallel with builds
	c.taskRunnerRoutinePool = routines.NewPool(c.taskRunners)

	c.dockerClient, err = docker.NewClient(log.StdLogger.Debugf)
	exitOnErr(err)

	s3Client, err := s3.NewClient(log.StdLogger)
	exitOnErr(err)
	c.uploader = baur.NewUploader(c.dockerClient, s3Client, filecopy.New(log.Debugf))

	c.vcsState = mustGetRepoState(repo.Path)

	if c.skipUpload {
		stdout.Printf("--skip-upload was passed, outputs won't be uploaded and task runs not recorded\n\n")
	}

	loader, err := baur.NewLoader(repo.Cfg, c.vcsState.CommitID, log.StdLogger)
	exitOnErr(err)

	tasks, err := loader.LoadTasks(args...)
	exitOnErr(err)

	baur.SortTasksByID(tasks)

	pendingTasks, err := c.filterPendingTasks(tasks)
	exitOnErr(err)

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
		ptCopy := pt
		c.taskRunnerRoutinePool.Queue(func() {
			task := ptCopy.task
			runResult := c.runTask(task)

			outputs, err := baur.OutputsFromTask(c.dockerClient, task)
			exitOnErrf(err, "%s", task.ID())

			if !outputsExist(task, outputs) {
				exitFunc(1)
			}

			if c.skipUpload {
				return
			}

			c.uploadRoutinePool.Queue(func() {
				// all outputs and uploads of the task are done in the same goroutine, serialized,
				// this is fine because we always only use a
				// uploadRoutinePool with 1 worker to run uploads in
				// parallel only with builds, uploads are not done in parallel
				c.uploadAndRecord(ctx, ptCopy.task, ptCopy.inputs, outputs, runResult)
			})
		})
	}

	c.taskRunnerRoutinePool.Wait()

	stdout.Println("all tasks executed, waiting for uploads to finish...")
	c.uploadRoutinePool.Wait()
	stdout.PrintSep()
	stdout.Printf("finished in: %s\n",
		term.FormatDuration(
			time.Since(startTime),
		),
	)
}

func (c *runCmd) runTask(task *baur.Task) *baur.RunResult {
	result, err := baur.NewTaskRunner().Run(task)
	exitOnErrf(err, "%s", task.ID())

	if result.Result.ExitCode != 0 {
		statusStr := term.RedHighlight("failed")

		stderr.Printf("%s: execution %s (%s), command exited with code %d, output:\n%s\n",
			task,
			statusStr,
			term.FormatDuration(
				result.StopTime.Sub(result.StartTime),
			),
			result.ExitCode,
			result.StrOutput())

		exitFunc(1)
	}

	statusStr := term.GreenHighlight("successful")

	stdout.TaskPrintf(task, "execution %s (%s)\n",
		statusStr,
		term.FormatDuration(
			result.StopTime.Sub(result.StartTime),
		),
	)

	return result
}

type pendingTask struct {
	task   *baur.Task
	inputs *baur.Inputs
}

func (c *runCmd) uploadAndRecord(
	ctx context.Context,
	task *baur.Task,
	inputs *baur.Inputs,
	outputs []baur.Output,
	runResult *baur.RunResult,
) {
	var uploadResults []*baur.UploadResult

	for _, output := range outputs {
		err := c.uploader.Upload(
			output,
			func(o baur.Output, info baur.UploadInfo) {
				log.Debugf("%s: uploading output %s to %s\n",
					task, output, info)
			},
			func(o baur.Output, result *baur.UploadResult) {
				size, err := o.SizeBytes()
				exitOnErrf(err, "%s: %s:", task.ID(), output)

				bps := uint64(math.Round(float64(size) / result.Stop.Sub(result.Start).Seconds()))

				stdout.TaskPrintf(task, "%s uploaded to %s (%s/s)\n",
					output, result.URL,
					term.FormatSize(bps),
				)

				uploadResults = append(uploadResults, result)
			},
		)

		exitOnErrf(err, "%s: %s", task.ID(), output)
	}

	id, err := baur.StoreRun(ctx, c.storage, c.vcsState, task, inputs, runResult, uploadResults)
	exitOnErrf(err, "%s", task.ID())

	stdout.TaskPrintf(task, "run stored in database with ID %s\n", term.Highlight(id))
}

func outputsExist(task *baur.Task, outputs []baur.Output) bool {
	allExist := true

	if len(outputs) == 0 {
		stdout.TaskPrintf(task, "does not produce outputs\n")
		return true
	}

	for _, output := range outputs {
		exists, err := output.Exists()
		exitOnErrf(err, "%s:", task.ID())

		if exists {
			size, err := output.SizeBytes()
			exitOnErrf(err, "%s:", task.ID())

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
		taskIDlen := len(task.ID())

		if taskIDlen > maxLen {
			maxLen = taskIDlen
		}
	}

	return maxLen
}

func (c *runCmd) filterPendingTasks(tasks []*baur.Task) ([]*pendingTask, error) {
	const sep = " => "

	taskIDColLen := maxTaskIDLen(tasks) + len(sep)
	statusEvaluator := baur.NewTaskStatusEvaluator(c.repoRootPath, c.storage, baur.NewCachingInputResolver(), c.inputStr, c.lookupInputStr)

	stdout.Printf("Evaluating status of tasks:\n\n")

	result := make([]*pendingTask, 0, len(tasks))
	for _, task := range tasks {
		status, inputs, run, err := statusEvaluator.Status(ctx, task)
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

		result = append(result, &pendingTask{
			task:   task,
			inputs: inputs,
		})
	}

	return result, nil
}
