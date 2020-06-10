package command

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur"
	"github.com/simplesurance/baur/git"
	"github.com/simplesurance/baur/internal/command/terminal"
	"github.com/simplesurance/baur/log"
	"github.com/simplesurance/baur/routines"
	"github.com/simplesurance/baur/storage"
	"github.com/simplesurance/baur/upload/docker"
	"github.com/simplesurance/baur/upload/filecopy"
	"github.com/simplesurance/baur/upload/s3"
)

// TODO remove support for setting docker username/passwd via env vars
const (
	dockerEnvUsernameVar = "BAUR_DOCKER_USERNAME"
	dockerEnvPasswordVar = "BAUR_DOCKER_PASSWORD"
)

func dockerClient() *docker.Client {
	var client *docker.Client
	var err error

	dockerUser, dockerPass := os.Getenv(dockerEnvUsernameVar), os.Getenv(dockerEnvPasswordVar)

	if len(dockerUser) != 0 {
		log.Debugf("using docker authentication data from %s, %s Environment variables, authenticating as '%s'",
			dockerEnvUsernameVar, dockerEnvPasswordVar, dockerUser)
		client, err = docker.NewClientwAuth(log.StdLogger.Debugf, dockerUser, dockerPass)
	} else {
		log.Debugf("environment variable %s not set", dockerEnvUsernameVar)
		client, err = docker.NewClient(log.StdLogger.Debugf)
	}

	exitOnErr(err)

	return client

}

func init() {
	rootCmd.AddCommand(&newRunCmd().Command)
}

type runCmd struct {
	cobra.Command

	// Cmdline parameters
	skipUpload bool
	force      bool

	// other fields
	storage      storage.Storer
	repoRootPath string
	dockerClient *docker.Client
	uploader     *baur.Uploader
	gitState     *git.RepositoryState

	uploadRoutinePool *routines.Pool
}

var runLongHelp = fmt.Sprintf(`
Execute tasks of applications.
By default all tasks of all applications with status %s and %s are run.

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
    %s
    %s
`,
	terminal.ColoredTaskStatus(baur.TaskStatusExecutionPending),
	terminal.ColoredTaskStatus(baur.TaskStatusInputsUndefined),

	terminal.Highlight(envVarPSQLURL),

	terminal.Highlight("AWS_REGION"),
	terminal.Highlight("AWS_ACCESS_KEY_ID"),
	terminal.Highlight("AWS_SECRET_ACCESS_KEY"),

	terminal.Highlight(dockerEnvUsernameVar),
	terminal.Highlight(dockerEnvPasswordVar),
	terminal.Highlight("DOCKER_HOST"),
	terminal.Highlight("DOCKER_API_VERSION"),
	terminal.Highlight("DOCKER_CERT_PATH"),
	terminal.Highlight("DOCKER_TLS_VERIFY"))

func newRunCmd() *runCmd {
	const example = `
run payment-service		run all tasks of the payment-service application and upload the produced outputs
run calc.check			run the check task of the calc application and upload the produced outputs
run --force			run and upload all tasks of applications, independent of their status
`

	cmd := runCmd{
		Command: cobra.Command{
			Use:     "run [<SPEC>|<PATH>]...",
			Short:   "run tasks",
			Long:    runLongHelp,
			Example: strings.TrimSpace(example),
		},
	}

	cmd.Run = cmd.run

	cmd.Flags().BoolVarP(&cmd.skipUpload, "skip-upload", "s", false,
		"skip uploading task outputs and recording the run")
	cmd.Flags().BoolVarP(&cmd.force, "force", "f", false,
		"enforce running tasks independent of their status")

	return &cmd
}

func (c *runCmd) run(cmd *cobra.Command, args []string) {
	startTime := time.Now()

	repo := MustFindRepository()
	c.repoRootPath = repo.Path

	c.storage = mustNewCompatibleStorage(repo)

	c.uploadRoutinePool = routines.NewPool(1) // run 1 upload in parallel with builds

	c.dockerClient = dockerClient()
	s3Client, err := s3.NewClient(log.StdLogger)
	exitOnErr(err)
	c.uploader = baur.NewUploader(c.dockerClient, s3Client, filecopy.New(log.Debugf))

	c.gitState = git.NewRepositoryState(repo.Path)

	if c.skipUpload {
		stdout.Printf("--skip-upload was passed, outputs won't be uploaded and task runs not recorded\n\n")
	}

	loader, err := baur.NewLoader(repo.Cfg, c.gitState.CommitID, log.StdLogger)
	exitOnErr(err)

	tasks, err := loader.LoadTasks(args...)
	exitOnErr(err)

	baur.SortTasksByID(tasks)

	pendingTasks, err := c.filterPendingTasks(tasks)
	exitOnErr(err)

	stdout.PrintSep()

	if c.force {
		stdout.Printf("Running %d/%d task(s) with status %s, %s\n\n",
			len(pendingTasks), len(tasks), terminal.ColoredTaskStatus(baur.TaskStatusExecutionPending), terminal.ColoredTaskStatus(baur.TaskStatusRunExist))
	} else {
		stdout.Printf("Running %d/%d task(s) with status %s\n\n",
			len(pendingTasks), len(tasks), terminal.ColoredTaskStatus(baur.TaskStatusExecutionPending))
	}

	c.runUploadStore(pendingTasks)
	stdout.Println("all tasks executed, waiting for uploads to finish...")
	c.uploadRoutinePool.Wait()
	stdout.PrintSep()
	stdout.Printf("finished in: %ss\n", strDurationSec(startTime, time.Now()))
}

type pendingTasks struct {
	task   *baur.Task
	inputs *baur.Inputs
}

type workerArg struct {
	ctx         context.Context
	pendingTask *pendingTasks
	outputs     []baur.Output
	runResult   *baur.RunResult
}

func (c *runCmd) uploadAndRecord(arg interface{}) {
	var uploadResults []*baur.UploadResult
	var ok bool

	wa, ok := arg.(*workerArg)
	if !ok {
		panic("upload worker got parameter of unsupported type")
	}

	for _, output := range wa.outputs {
		err := c.uploader.Upload(
			output,
			func(o baur.Output, info baur.UploadInfo) {
				log.Debugf("%s: uploading output %s to %s\n",
					wa.pendingTask.task, output, info)
			},

			func(o baur.Output, result *baur.UploadResult) {
				size, err := o.Size()
				exitOnErr(err)

				mbps := float64(size) / 1024 / 1024 / result.Stop.Sub(result.Start).Seconds()

				stdout.TaskPrintf(wa.pendingTask.task, "%s uploaded to %s (%.3f MiB/s)\n",
					output, result.URL,
					mbps)

				uploadResults = append(uploadResults, result)
			},
		)

		exitOnErr(err)
	}

	id, err := baur.StoreRun(ctx, c.storage, c.gitState, wa.pendingTask.task, wa.pendingTask.inputs, wa.runResult, uploadResults)
	exitOnErr(err)

	stdout.TaskPrintf(wa.pendingTask.task, "run stored in database with ID %s\n", terminal.Highlight(id))
}

func (c *runCmd) runUploadStore(taskToRun []*pendingTasks) {
	taskRunner := baur.NewTaskRunner()

	for _, t := range taskToRun {
		runResult, err := taskRunner.Run(t.task)
		exitOnErr(err)

		if runResult.Result.ExitCode != 0 {
			statusStr := terminal.RedHighlight("failed")

			log.Fatalf("%s: execution %s (%ss), command exited with code %d, output:\n%s\n",
				t.task,
				statusStr,
				strDurationSec(runResult.StartTime, runResult.StopTime),
				runResult.ExitCode,
				runResult.StrOutput())
		}

		statusStr := terminal.GreenHighlight("successful")

		stdout.TaskPrintf(t.task, "execution %s (%ss)\n",
			statusStr,
			strDurationSec(runResult.StartTime, runResult.StopTime),
		)

		outputs, err := baur.OutputsFromTask(c.dockerClient, t.task)
		exitOnErr(err)

		if !outputsExit(t.task, outputs) {
			os.Exit(1)
		}

		if c.skipUpload {
			continue
		}

		c.uploadRoutinePool.Queue(c.uploadAndRecord, &workerArg{
			ctx:         ctx,
			pendingTask: t,
			outputs:     outputs,
			runResult:   runResult,
		})

	}
}

func outputsExit(task *baur.Task, outputs []baur.Output) bool {
	allExist := true

	if len(outputs) == 0 {
		stdout.TaskPrintf(task, "does not produce outputs\n")
		return true
	}

	for _, output := range outputs {
		exists, err := output.Exists()
		exitOnErrf(err, "%s :", task.ID())

		if exists {
			size, err := output.Size()
			exitOnErrf(err, "%s :", task.ID())

			stdout.TaskPrintf(task, "created %s (size: %s MiB)\n",
				output, bytesToMib(size))

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

func (c *runCmd) filterPendingTasks(tasks []*baur.Task) ([]*pendingTasks, error) {
	var result []*pendingTasks
	const sep = " => "

	taskIDColLen := maxTaskIDLen(tasks) + len(sep)
	statusEvaluator := baur.NewTaskStatusEvaluator(c.repoRootPath, c.storage, baur.NewInputResolver())

	stdout.Printf("Evaluating status of tasks:\n\n")

	for _, task := range tasks {
		status, inputs, run, err := statusEvaluator.Status(ctx, task)
		if err != nil {
			return nil, fmt.Errorf("%s: evaluating task status failed: %w", task, err)
		}

		if status == baur.TaskStatusRunExist {
			stdout.Printf("%-*s%s%s (%s)\n",
				taskIDColLen, task, sep, terminal.ColoredTaskStatus(status), terminal.GreenHighlight(run.ID))

			if !c.force {
				continue
			}
		} else {
			stdout.Printf("%-*s%s%s\n", taskIDColLen, task, sep, terminal.ColoredTaskStatus(status))
		}

		result = append(result, &pendingTasks{
			task:   task,
			inputs: inputs,
		})
	}

	return result, nil
}
