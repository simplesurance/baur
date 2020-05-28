package command

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur"
	"github.com/simplesurance/baur/git"
	"github.com/simplesurance/baur/log"
	"github.com/simplesurance/baur/storage"
	"github.com/simplesurance/baur/term"
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
	if err != nil {
		log.Fatalln(err)
	}

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
	// TODO: do not store docker.Client 2x times, it's also in uploader
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
	coloredTaskStatus(baur.TaskStatusExecutionPending),
	coloredTaskStatus(baur.TaskStatusInputsUndefined),

	highlight(envVarPSQLURL),

	highlight("AWS_REGION"),
	highlight("AWS_ACCESS_KEY_ID"),
	highlight("AWS_SECRET_ACCESS_KEY"),

	highlight(dockerEnvUsernameVar),
	highlight(dockerEnvPasswordVar),
	highlight("DOCKER_HOST"),
	highlight("DOCKER_API_VERSION"),
	highlight("DOCKER_CERT_PATH"),
	highlight("DOCKER_TLS_VERIFY"))

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

	term.PrintSep()

	if c.force {
		fmt.Printf("Running %d/%d task(s) with status %s, %s\n",
			len(pendingTasks), len(tasks), coloredTaskStatus(baur.TaskStatusExecutionPending), coloredTaskStatus(baur.TaskStatusRunExist))
	} else {
		fmt.Printf("Running %d/%d task(s) with status %s\n",
			len(pendingTasks), len(tasks), coloredTaskStatus(baur.TaskStatusExecutionPending))
	}

	c.runUploadStore(pendingTasks)
}

func (c *runCmd) runUploadStore(taskToRun []*taskWithInputs) {
	taskRunner := baur.NewTaskRunner()

	for _, t := range taskToRun {
		fmt.Printf("%s: execution started\n", t.task)

		runResult, err := taskRunner.Run(t.task)
		exitOnErr(err)

		if runResult.Result.ExitCode == 0 {
			statusStr := greenHighlight("successful")

			fmt.Printf("%s: execution %s (%ss)\n",
				t.task,
				statusStr,
				strDurationSec(runResult.StartTime, runResult.StopTime),
			)
		} else {
			statusStr := redHighlight("failed")

			// TODO: make this only fatal if a --fatal or so commandline flag is passed
			log.Fatalf("%s: execution %s (%ss), command exited with code %d, output:\n%s\n",
				t.task,
				statusStr,
				strDurationSec(runResult.StartTime, runResult.StopTime),
				runResult.ExitCode,
				runResult.StrOutput())
		}

		// TODO: Create a ConsolePrinter object to log to the console, which
		// adds prefixes and maybe also has a print function that accepts a
		// task parameter and prints a prefix for it?

		outputs, err := baur.OutputsFromTask(c.dockerClient, t.task)
		exitOnErr(err)

		// TODO: make this only fatal if a --fatal or so commandline flag is passed
		if !outputsExit(t.task, outputs) {
			os.Exit(1)
		}

		// TODO: make uploading and storing outputs asynchronous
		uploadResults := c.uploadOutputs(t.task, outputs)

		id, err := baur.StoreRun(ctx, c.storage, c.gitState, t.task, t.inputs, runResult, uploadResults)
		exitOnErr(err)

		fmt.Printf("%s: run stored in database with ID %s\n", t.task, highlight(id))
	}
}

func outputsExit(task *baur.Task, outputs []baur.Output) bool {
	allExist := true

	if len(outputs) == 0 {
		fmt.Printf("%s: does not produce outputs\n", task)
		return true
	}

	for _, output := range outputs {
		exists, err := output.Exists()
		exitOnErrf(err, "%s :", task.ID())

		if exists {
			size, err := output.Size()
			exitOnErrf(err, "%s :", task.ID())

			fmt.Printf("%s: created output %s (size: %s MiB)\n",
				task, output, bytesToMib(size))

			continue
		}

		allExist = false
		// TODO: use printf or a streams struct?
		log.Errorf("%s: did not created output %s", task, output)
	}

	return allExist
}

func (c *runCmd) uploadOutputs(task *baur.Task, outputs []baur.Output) []*baur.UploadResult {
	// TODO: ensure we handle recording tasks without outputs correctly and also record it to the db
	var uploadResults []*baur.UploadResult

	for _, output := range outputs {
		err := c.uploader.Upload(
			output,
			func(o baur.Output, info baur.UploadInfo) {
				fmt.Printf("%s: uploading output %s to %s\n",
					task, output, info)
			},
			// TODO: log upload speed
			func(o baur.Output, uploadResult *baur.UploadResult) {
				fmt.Printf("%s: uploaded %s to %s (%ss)\n",
					task, output, uploadResult.URL,
					strDurationSec(uploadResult.Start, uploadResult.Stop))

				uploadResults = append(uploadResults, uploadResult)
			},
		)
		// TODO: make this only fatal if a --fatal or so commandline flag is passed
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

	fmt.Printf("Evaluating status of tasks:\n\n")

	for _, task := range tasks {
		status, inputs, run, err := statusEvaluator.Status(ctx, task)
		if err != nil {
			return nil, fmt.Errorf("%s: evaluating task status failed: %w", task, err)
		}

		if status == baur.TaskStatusRunExist {
			fmt.Printf("%-*s%s%s (%s)\n",
				taskIDColLen, task, sep, coloredTaskStatus(status), greenHighlight(run.ID))

			if !c.force {
				continue
			}
		}

		fmt.Printf("%-*s%s%s\n", taskIDColLen, task, sep, coloredTaskStatus(status))

		result = append(result, &taskWithInputs{
			task:   task,
			inputs: inputs,
		})
	}

	return result, nil
}
