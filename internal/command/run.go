package command

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur/v1"
	"github.com/simplesurance/baur/v1/internal/command/term"
	"github.com/simplesurance/baur/v1/internal/log"
	"github.com/simplesurance/baur/v1/internal/routines"
	"github.com/simplesurance/baur/v1/internal/upload/docker"
	"github.com/simplesurance/baur/v1/internal/upload/filecopy"
	"github.com/simplesurance/baur/v1/internal/upload/s3"
	"github.com/simplesurance/baur/v1/internal/vcs"
	"github.com/simplesurance/baur/v1/storage"
)

const runExample = `
baur run auth				run all tasks of the auth application, upload the produced outputs
baur run calc.check			run the check task of the calc application and upload the produced outputs
baur run --force			run and upload all tasks of applications, independent of their status
`

var runLongHelp = fmt.Sprintf(`
Execute tasks of applications.
By default all tasks of all applications with status %s are run.

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
	inputStr       string
	lookupInputStr string

	// other fields
	storage      storage.Storer
	repoRootPath string
	dockerClient *docker.Client
	uploader     *baur.Uploader
	vcsState     vcs.StateFetcher

	uploadRoutinePool *routines.Pool
}

func newRunCmd() *runCmd {
	cmd := runCmd{
		Command: cobra.Command{
			Use:     "run [<SPEC>|<PATH>]...",
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
	cmd.Flags().StringVar(&cmd.inputStr, "input-str", "",
		"include a string as an input")
	cmd.Flags().StringVar(&cmd.lookupInputStr, "lookup-input-str", "",
		"if a run can not be found, try to find a run with this value as input-string")

	return &cmd
}

func (c *runCmd) run(cmd *cobra.Command, args []string) {
	var err error

	startTime := time.Now()

	repo := mustFindRepository()
	c.repoRootPath = repo.Path

	c.storage = mustNewCompatibleStorage(repo)

	c.uploadRoutinePool = routines.NewPool(1) // run 1 upload in parallel with builds

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

	c.runUploadStore(pendingTasks)
	stdout.Println("all tasks executed, waiting for uploads to finish...")
	c.uploadRoutinePool.Wait()
	stdout.PrintSep()
	stdout.Printf("finished in: %s\n",
		term.FormatDuration(
			time.Since(startTime),
		),
	)
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
				size, err := o.Size()
				exitOnErrf(err, "%s: %s:", task.ID(), output)

				term.FormatDuration(
					result.Stop.Sub(result.Start),
				)
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

func (c *runCmd) runUploadStore(taskToRun []*pendingTask) {
	taskRunner := baur.NewTaskRunner()

	for _, t := range taskToRun {
		// TODO: record the result as failed if run exitCode is != 0
		// except when a flag like --errors-are-fatal is passed
		runResult, err := taskRunner.Run(t.task)
		exitOnErrf(err, "%s", t.task.ID())

		if runResult.Result.ExitCode != 0 {
			statusStr := term.RedHighlight("failed")

			log.Fatalf("%s: execution %s (%s), command exited with code %d, output:\n%s\n",
				t.task,
				statusStr,
				term.FormatDuration(
					runResult.StopTime.Sub(runResult.StartTime),
				),
				runResult.ExitCode,
				runResult.StrOutput())
		}

		statusStr := term.GreenHighlight("successful")

		stdout.TaskPrintf(t.task, "execution %s (%s)\n",
			statusStr,
			term.FormatDuration(
				runResult.StopTime.Sub(runResult.StartTime),
			),
		)

		outputs, err := baur.OutputsFromTask(c.dockerClient, t.task)
		exitOnErrf(err, "%s", t.task.ID())

		if !outputsExist(t.task, outputs) {
			exitFunc(1)
		}

		if c.skipUpload {
			continue
		}

		// copy the iteration variable, to prevent that it's value
		// changes in the closure to 't' of the next iteration before
		// the closure is executed
		taskCopy := t
		c.uploadRoutinePool.Queue(func() {
			c.uploadAndRecord(ctx, taskCopy.task, taskCopy.inputs, outputs, runResult)
		})
	}
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
			size, err := output.Size()
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
	var result []*pendingTask
	const sep = " => "

	taskIDColLen := maxTaskIDLen(tasks) + len(sep)
	statusEvaluator := baur.NewTaskStatusEvaluator(c.repoRootPath, c.storage, baur.NewInputResolver(), c.inputStr, c.lookupInputStr)

	stdout.Printf("Evaluating status of tasks:\n\n")

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
