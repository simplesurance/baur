package command

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/simplesurance/baur"
	"github.com/simplesurance/baur/build"
	"github.com/simplesurance/baur/build/seq"
	"github.com/simplesurance/baur/git"
	"github.com/simplesurance/baur/log"
	"github.com/simplesurance/baur/prettyprint"
	"github.com/simplesurance/baur/storage"
	"github.com/simplesurance/baur/term"
	"github.com/simplesurance/baur/upload/docker"
	"github.com/simplesurance/baur/upload/filecopy"
	"github.com/simplesurance/baur/upload/s3"
	"github.com/simplesurance/baur/upload/scheduler"
	sequploader "github.com/simplesurance/baur/upload/scheduler/seq"
)

const (
	dockerEnvUsernameVar = "BAUR_DOCKER_USERNAME"
	dockerEnvPasswordVar = "BAUR_DOCKER_PASSWORD"

	appColSep = " => "
	sepLen    = len(appColSep)
)

var buildLongHelp = fmt.Sprintf(`
Execute the 'build' task of applications.
If no path or application name is passed, the build task of all applications in the repository are run.
By default only applications with status %s and %s are build.

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

const buildExampleHelp = `
build payment-service		build and upload the application with the name payment-service
build --verbose --force		rebuild and upload all applications, enable verbose output
build --skip-upload shop-ui	build the application with the name shop-ui, skip uploading it's build ouputs
build ui/shop			build and upload the application in the directory ui/shop
`

var buildCmd = &cobra.Command{
	Use:     "build [<PATH>|<APP-NAME>]...",
	Short:   "build applications",
	Long:    strings.TrimSpace(buildLongHelp),
	Run:     buildRun,
	Example: strings.TrimSpace(buildExampleHelp),
	Args:    cobra.ArbitraryArgs,
}

var (
	buildSkipUpload bool
	buildForce      bool

	result     = map[string]*storage.TaskRunFull{}
	resultLock = sync.Mutex{}

	store          storage.Storer
	outputBackends baur.BuildOutputBackends
)

type uploadUserData struct {
	App    *baur.App
	Output baur.BuildOutput
}

type buildUserData struct {
	App              *baur.App
	Inputs           *baur.Inputs
	TotalInputDigest string
}

func init() {
	buildCmd.Flags().BoolVarP(&buildSkipUpload, "skip-upload", "s", false,
		"skip uploading build outputs and recording the build")
	buildCmd.Flags().BoolVarP(&buildForce, "force", "f", false,
		"force rebuilding of all applications")
	rootCmd.AddCommand(buildCmd)
}

func resultAddBuildResult(repo *baur.Repository, bud *buildUserData, r *build.Result, gitCommitID string, gitWorktreeIsDirty bool) {
	resultLock.Lock()
	defer resultLock.Unlock()

	totalInputDigest, err := bud.Inputs.Digest()
	if err != nil {
		panic(fmt.Sprintf(
			"%s: retrieving totalInputDigest from Inputs() failed, "+
				"it should have returned the precalculated digest %s",
			bud.App.Task(),
			err,
		))
	}

	b := storage.TaskRunFull{
		TaskRun: storage.TaskRun{
			// TODO: store the real task name, instead of always build,
			TaskName:         "build",
			ApplicationName:  bud.App.Name,
			VCSIsDirty:       gitWorktreeIsDirty,
			VCSRevision:      gitCommitID,
			StartTimestamp:   r.StartTs,
			StopTimestamp:    r.StopTs,
			Result:           storage.ResultSuccess,
			TotalInputDigest: totalInputDigest.String(),
		},
		Inputs: taskInputsToStorageInputs(bud.Inputs),
	}

	result[bud.App.Name] = &b

}

func resultAddUploadResult(appName string, ar baur.BuildOutput, r *scheduler.Result) {
	var arType storage.ArtifactType
	var uploadMethod storage.UploadMethod

	resultLock.Lock()
	defer resultLock.Unlock()

	b, exist := result[appName]
	if !exist {
		log.Fatalf("resultAddUploadResult: %q does not exist in build result map", appName)
	}

	switch r.Job.Type() {
	case scheduler.JobDocker:
		arType = storage.ArtifactTypeDocker
		uploadMethod = storage.UploadMethodDockerRegistry
	case scheduler.JobFileCopy:
		arType = storage.ArtifactTypeFile
		uploadMethod = storage.UploadMethodFileCopy
	case scheduler.JobS3:
		arType = storage.ArtifactTypeFile
		uploadMethod = storage.UploadMethodS3
	default:
		panic(fmt.Sprintf("unknown job type %v", r.Job.Type()))
	}

	artDigest, err := ar.Digest()
	if err != nil {
		log.Fatalf("getting digest for output %q failed: %s", ar, err)
	}

	arSize, err := ar.Size(&outputBackends)
	if err != nil {
		log.Fatalf("getting size of output %q failed: %s", ar, err)
	}

	b.Outputs = append(b.Outputs, &storage.Output{
		Name:      ar.Name(),
		Type:      arType,
		SizeBytes: uint64(arSize),
		Uploads: []*storage.Upload{
			{
				URI:                  r.URL,
				Method:               uploadMethod,
				UploadStartTimestamp: r.Start,
				UploadStopTimestamp:  r.Stop,
			},
		},
		Digest: artDigest.String(),
	})
}

func recordResultIsComplete(task *baur.Task) (bool, *storage.TaskRunFull) {
	resultLock.Lock()
	defer resultLock.Unlock()

	b, exist := result[task.AppName]
	if !exist {
		log.Fatalf("recordResultIfComplete: %q does not exist in build result map", task.AppName)
	}

	if taskOutputCount(task) == len(b.Outputs) {
		return true, b
	}

	return false, nil
}

func taskOutputCount(task *baur.Task) int {
	return len(task.Outputs.DockerImage) + len(task.Outputs.File)
}

func outputCount(jobs []*build.Job) int {
	var cnt int

	for _, j := range jobs {
		bud := j.UserData.(*buildUserData)
		cnt += taskOutputCount(bud.App.Task())
	}

	return cnt
}

func dockerAuthFromEnv() (string, string) {
	return os.Getenv(dockerEnvUsernameVar), os.Getenv(dockerEnvPasswordVar)
}

func startBGUploader(outputCnt int, uploadChan chan *scheduler.Result) scheduler.Manager {
	var dockerUploader *docker.Client
	s3Uploader, err := s3.NewClient(log.StdLogger)
	if err != nil {
		log.Fatalln(err.Error())
	}

	dockerUser, dockerPass := dockerAuthFromEnv()
	if len(dockerUser) != 0 {
		log.Debugf("using docker authentication data from %s, %s Environment variables, authenticating as '%s'",
			dockerEnvUsernameVar, dockerEnvPasswordVar, dockerUser)
		dockerUploader, err = docker.NewClientwAuth(log.StdLogger.Debugf, dockerUser, dockerPass)
	} else {
		log.Debugf("environment variable %s not set", dockerEnvUsernameVar)
		dockerUploader, err = docker.NewClient(log.StdLogger.Debugf)
	}
	if err != nil {
		log.Fatalln(err)
	}

	filecopyUploader := filecopy.New(log.Debugf)

	uploader := sequploader.New(log.StdLogger, filecopyUploader, s3Uploader, dockerUploader, uploadChan)

	outputBackends.DockerClt = dockerUploader

	go uploader.Start()

	return uploader
}

func waitPrintUploadStatus(uploader scheduler.Manager, uploadChan chan *scheduler.Result, finished chan struct{}, outputCnt int) {
	var resultCnt int

	for res := range uploadChan {
		ud, ok := res.Job.GetUserData().(*uploadUserData)
		if !ok {
			log.Fatalln("upload result user data has unexpected type")
		}

		if res.Err != nil {
			log.Fatalf("upload of %q failed: %s\n", ud.Output, res.Err)
		}

		fmt.Printf("%s: %s uploaded to %s (%ss)\n",
			ud.App.Name, ud.Output.LocalPath(), res.URL,
			durationToStrSeconds(res.Stop.Sub(res.Start)))

		resultAddUploadResult(ud.App.Name, ud.Output, res)

		complete, taskRun := recordResultIsComplete(ud.App.Task())
		if complete {
			log.Debugf("%s: storing build information in database\n", ud.App)
			id, err := store.SaveTaskRun(ctx, taskRun)
			if err != nil {
				log.Fatalf("%s: storing task run information failed: %s", ud.App.Name, err)
			}
			fmt.Printf("%s: build %d stored in database\n", ud.App.Name, id)

			log.Debugf("stored the following build information: %s\n", prettyprint.AsString(taskRun))
		}

		resultCnt++
		if resultCnt == outputCnt {
			break
		}
	}

	uploader.Stop()

	close(finished)
}

func maxAppNameLen(apps []*baur.App) int {
	var maxLen int

	for _, app := range apps {
		if len(app.Name) > maxLen {
			maxLen = len(app.Name)
		}
	}

	return maxLen
}

func taskInputsToStorageInputs(inputs *baur.Inputs) []*storage.Input {
	result := make([]*storage.Input, len(inputs.Files))

	for i, input := range inputs.Files {
		digest, err := input.Digest()
		if err != nil {
			panic(fmt.Sprintf(
				"%s: retrieving digest of input failed, "+
					"it should have returned the precalculated digest %s",
				input,
				err,
			))
		}

		result[i] = &storage.Input{
			Digest: digest.String(),
			URI:    input.RepoRelPath(),
		}
	}

	return result
}

func pendingBuilds(storer storage.Storer, apps []*baur.App, repositoryRootDir string, rebuildExisting bool) []*build.Job {
	var res []*build.Job
	statusMgr := baur.NewTaskStatusEvaluator(repositoryRootDir, storer, baur.NewInputResolver())

	appNameColLen := maxAppNameLen(apps) + sepLen

	for _, app := range apps {
		task := app.Task()

		log.Debugf("%s: resolving build inputs and calculating digests...", task)
		status, inputs, run, err := statusMgr.Status(ctx, task)
		exitOnErrf(err, task.String())

		if status == baur.TaskStatusUndefined {
			panic(fmt.Sprintf("task status is %s and err is not nil (%s) \n", status, err))
		}

		if status == baur.TaskStatusRunExist {
			fmt.Printf("%-*s%s%s (%s)\n",
				appNameColLen, app.Name, appColSep, coloredTaskStatus(status), highlight(run.ID))
		} else {
			fmt.Printf("%-*s%s%s\n",
				appNameColLen, app.Name, appColSep, coloredTaskStatus(status))
		}

		if status == baur.TaskStatusInputsUndefined {
			continue
		}

		if !rebuildExisting && status == baur.TaskStatusRunExist {
			continue
		}

		res = append(res, &build.Job{
			Application: app.Name,
			Directory:   app.Path,
			Command:     app.Task().Command,
			UserData: &buildUserData{
				App:    app,
				Inputs: inputs,
			},
		})
	}

	return res
}

func buildRun(cmd *cobra.Command, args []string) {
	var apps []*baur.App
	var uploadWatchFin chan struct{}
	var uploader scheduler.Manager
	var buildJobs []*build.Job

	repo := MustFindRepository()
	store = mustNewCompatibleStorage(repo)

	apps = mustArgToApps(repo, args)
	baur.SortAppsByName(apps)

	startTs := time.Now()

	fmt.Printf("Evaluating build status of applications:\n")
	buildJobs = pendingBuilds(store, apps, repo.Path, buildForce)

	if buildForce {
		fmt.Printf("\nBuilding applications with build status: %s or %s\n",
			coloredTaskStatus(baur.TaskStatusExecutionPending), coloredTaskStatus(baur.TaskStatusRunExist))
	} else {
		fmt.Printf("\nBuilding applications with build status: %s\n",
			coloredTaskStatus(baur.TaskStatusExecutionPending))
	}

	if buildSkipUpload {
		fmt.Println("Outputs are not uploaded.")
	}

	if len(apps) == 0 {
		term.PrintSep()

		if !buildForce {
			fmt.Println("If you want to rebuild applications pass '-f' to 'baur build'.")
		}

		os.Exit(0)
	}

	buildChan := make(chan *build.Result, len(buildJobs))
	builder := seq.New(buildJobs, buildChan)
	outputCnt := outputCount(buildJobs)

	if !buildSkipUpload {
		uploadChan := make(chan *scheduler.Result, outputCnt)
		uploader = startBGUploader(outputCnt, uploadChan)
		uploadWatchFin = make(chan struct{}, 1)
		go waitPrintUploadStatus(uploader, uploadChan, uploadWatchFin, outputCnt)

	}

	term.PrintSep()

	go builder.Start()

	gitCommitID, err := git.CommitID(repo.Path)
	if err != nil {
		log.Fatalf("determining Git commit ID failed, "+
			"ensure that the git command is in a directory in $PATH and "+
			"that the .baur.toml file is part of a git repository.\n%s", err.Error())
	}

	gitWorktreeIsDirty, err := git.WorktreeIsDirty(repo.Path)
	if err != nil {
		log.Fatalf("determining Git worktree state failed, "+
			"ensure that the git command is in a directory in $PATH and "+
			"that the .baur.toml file is part of a git repository.\n%s", err.Error())

	}

	for status := range buildChan {
		bud := status.Job.UserData.(*buildUserData)
		app := bud.App

		if status.Error != nil {
			log.Fatalf("%s: build failed: %s", app.Name, status.Error)
		}

		if status.ExitCode != 0 {
			log.Fatalf("%s: build failed: command (%q) exited with code %d "+
				"Output: %s",
				app.Name, status.Job.Command, status.ExitCode, status.Output)
		}

		fmt.Printf("%s: build successful (%.3fs)\n", app.Name, status.StopTs.Sub(status.StartTs).Seconds())
		resultAddBuildResult(repo, bud, status, gitCommitID, gitWorktreeIsDirty)

		task := app.Task()
		outputs, err := task.BuildOutputs()
		if err != nil {
			log.Fatalln(err)
		}

		for _, ar := range outputs {
			if !ar.Exists() {
				log.Fatalf("%s: build output %q did not exist after build",
					app, ar)
			}

			if !buildSkipUpload {
				uj, err := ar.UploadJob()
				if err != nil {
					log.Fatalf("%s: could not get upload job for build output %s: %s",
						app, ar, err)
				}

				uj.SetUserData(&uploadUserData{
					App:    app,
					Output: ar,
				})

				uploader.Add(uj)

			}
			d, err := ar.Digest()
			if err != nil {
				log.Fatalf("%s: calculating input digest of %s failed: %s",
					app.Name, ar, err)
			}

			fmt.Printf("%s: created %s (%s)\n", app.Name, ar, d)
		}

	}

	if !buildSkipUpload && outputCnt > 0 {
		fmt.Println("waiting for uploads to finish...")
		<-uploadWatchFin
	}

	if len(buildJobs) > 0 {
		term.PrintSep()
	}
	fmt.Printf("finished in %ss\n", durationToStrSeconds(time.Since(startTs)))
}
