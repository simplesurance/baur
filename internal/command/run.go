package command

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur"
	"github.com/simplesurance/baur/git"
	"github.com/simplesurance/baur/internal/command/terminal"
	"github.com/simplesurance/baur/log"
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

type taskWithInputs struct {
	task   *baur.Task
	inputs *baur.Inputs
}

func (c *runCmd) run(cmd *cobra.Command, args []string) {
	repo := MustFindRepository()
	c.repoRootPath = repo.Path

	c.storage = mustNewCompatibleStorage(repo)
	c.dockerClient = dockerClient()

	// TODO: only create clients if needed to prevent failures because of them when they are not used
	s3Client, err := s3.NewClient(log.StdLogger)
	exitOnErr(err)

	c.uploader = baur.NewUploader(c.dockerClient, s3Client, filecopy.New(log.Debugf))

	c.gitState = git.NewRepositoryState(repo.Path)

	loader, err := baur.NewLoader(repo.Cfg, c.gitState.CommitID, log.StdLogger)
	exitOnErr(err)

	tasks, err := loader.LoadTasks(args...)
	exitOnErr(err)

	baur.SortTasksByID(tasks)

	pendingTasks, err := c.filterPendingTasks(tasks)
	exitOnErr(err)

	stdout.PrintSep()

	if c.force {
		stdout.Printf("Running %d/%d task(s) with status %s, %s\n",
			len(pendingTasks), len(tasks), terminal.ColoredTaskStatus(baur.TaskStatusExecutionPending), terminal.ColoredTaskStatus(baur.TaskStatusRunExist))
	} else {
		stdout.Printf("Running %d/%d task(s) with status %s\n",
			len(pendingTasks), len(tasks), terminal.ColoredTaskStatus(baur.TaskStatusExecutionPending))
	}

	c.runUploadStore(pendingTasks)
}

func (c *runCmd) runUploadStore(taskToRun []*taskWithInputs) {
	taskRunner := baur.NewTaskRunner()

	for _, t := range taskToRun {
		stdout.Printf("%s: execution started\n", t.task)

		runResult, err := taskRunner.Run(t.task)
		exitOnErr(err)

		if runResult.Result.ExitCode == 0 {
			statusStr := terminal.GreenHighlight("successful")

			stdout.TaskPrintf(t.task, "execution %s (%ss)\n",
				statusStr,
				strDurationSec(runResult.StartTime, runResult.StopTime),
			)
		} else {
			statusStr := terminal.RedHighlight("failed")

			log.Fatalf("%s: execution %s (%ss), command exited with code %d, output:\n%s\n",
				t.task,
				statusStr,
				strDurationSec(runResult.StartTime, runResult.StopTime),
				runResult.ExitCode,
				runResult.StrOutput())
		}

		outputs, err := baur.OutputsFromTask(c.dockerClient, t.task)
		exitOnErr(err)

		if !outputsExit(t.task, outputs) {
			os.Exit(1)
		}

		// TODO: make uploading and storing outputs asynchronous
		uploadResults := c.uploadOutputs(t.task, outputs)

		id, err := baur.StoreRun(ctx, c.storage, c.gitState, t.task, t.inputs, runResult, uploadResults)
		exitOnErr(err)

		stdout.TaskPrintf(t.task, "run stored in database with ID %s\n", terminal.Highlight(id))
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

			stdout.TaskPrintf(task, "created output %s (size: %s MiB)\n",
				output, bytesToMib(size))

			continue
		}

		allExist = false
		stderr.TaskPrintf(task, "has %s as output defined but it was not created by the task run\n", output)
	}

	return allExist
}

func (c *runCmd) uploadOutputs(task *baur.Task, outputs []baur.Output) []*baur.UploadResult {
	var uploadResults []*baur.UploadResult

	for _, output := range outputs {
		err := c.uploader.Upload(
			output,
			func(o baur.Output, info baur.UploadInfo) {
				stdout.TaskPrintf(task, "uploading output %s to %s\n",
					output, info)
			},
			// TODO: log upload speed
			func(o baur.Output, uploadResult *baur.UploadResult) {
				stdout.TaskPrintf(task, "uploaded %s to %s (%ss)\n",
					output, uploadResult.URL,
					strDurationSec(uploadResult.Start, uploadResult.Stop))

				uploadResults = append(uploadResults, uploadResult)
			},
		)

		exitOnErr(err)
	}

	return uploadResults
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

func (c *runCmd) filterPendingTasks(tasks []*baur.Task) ([]*taskWithInputs, error) {
	var result []*taskWithInputs
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

		result = append(result, &taskWithInputs{
			task:   task,
			inputs: inputs,
		})
	}

	return result, nil
}
