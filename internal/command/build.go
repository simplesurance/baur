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
	coloredBuildStatus(baur.BuildStatusPending),
	coloredBuildStatus(baur.BuildStatusInputsUndefined),

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
build --branch=shop shop-ui	build the shop-ui using the shop branch identifier
build --branch=shop --compare=master build the shop-u using the shop branch identifier.  If no builds exist with shop will use master instead
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

	branchFlag  string
	compareFlag string

	result     = map[string]*storage.Build{}
	resultLock = sync.Mutex{}

	store          storage.Storer
	outputBackends baur.BuildOutputBackends
)

type uploadUserData struct {
	App    *baur.App
	Output baur.BuildOutput
}

type buildUserData struct {
	// TODO: change App to Task
	App              *baur.App
	Inputs           []*storage.Input
	TotalInputDigest string
}

func init() {
	buildCmd.Flags().BoolVarP(&buildSkipUpload, "skip-upload", "s", false,
		"skip uploading build outputs and recording the build")
	buildCmd.Flags().BoolVarP(&buildForce, "force", "f", false,
		"force rebuilding of all applications")
	buildCmd.PersistentFlags().StringVarP(&branchFlag, "branch", "B", "default",
		"branch identifier to store build against")
	buildCmd.PersistentFlags().StringVarP(&compareFlag, "compare", "c", "",
		"Branch identifier to fall back to if no builds exist")
	rootCmd.AddCommand(buildCmd)
}

func resultAddBuildResult(repo *baur.Repository, bud *buildUserData, r *build.Result, gitCommitID string, gitWorktreeIsDirty bool) {
	resultLock.Lock()
	defer resultLock.Unlock()

	b := storage.Build{
		Application: storage.Application{Name: bud.App.Name},
		VCSState: storage.VCSState{
			CommitID: gitCommitID,
			IsDirty:  gitWorktreeIsDirty,
		},
		StartTimeStamp:   r.StartTs,
		StopTimeStamp:    r.StopTs,
		Branch:           branchFlag,
		Inputs:           bud.Inputs,
		TotalInputDigest: bud.TotalInputDigest,
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
		arType = storage.DockerArtifact
		uploadMethod = storage.DockerRegistry
	case scheduler.JobFileCopy:
		arType = storage.FileArtifact
		uploadMethod = storage.FileCopy
	case scheduler.JobS3:
		arType = storage.FileArtifact
		uploadMethod = storage.S3
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
		SizeBytes: arSize,
		Type:      arType,
		Upload: storage.Upload{
			URI:            r.URL,
			Method:         uploadMethod,
			UploadDuration: r.Duration,
		},
		Digest: artDigest.String(),
	})
}

func recordResultIsComplete(task *baur.Task) (bool, *storage.Build) {
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
			ud.App.Name, ud.Output.LocalPath(), res.URL, durationToStrSeconds(res.Duration))

		resultAddUploadResult(ud.App.Name, ud.Output, res)

		complete, build := recordResultIsComplete(ud.App.Task())
		if complete {
			log.Debugf("%s: storing build information in database\n", ud.App)
			if err := store.Save(build); err != nil {
				log.Fatalf("storing build information about %q failed: %s", ud.App.Name, err)
			}
			fmt.Printf("%s: build %d stored in database\n", ud.App.Name, build.ID)

			log.Debugf("stored the following build information: %s\n", prettyprint.AsString(build))
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

func filesToStorageInputs(inputs *baur.Inputs) ([]*storage.Input, error) {
	result := make([]*storage.Input, len(inputs.Files))

	for i, file := range inputs.Files {
		digest, err := file.Digest()
		if err != nil {
			// should never happen because the digest is already calculated before and here file should only return the stored value
			return nil, err
		}

		result[i] = &storage.Input{
			Digest: digest.String(),
			URI:    file.RepoRelPath(),
		}
	}

	return result, nil
}

func pendingBuilds(storer storage.Storer, apps []*baur.App, repositoryRootDir string, rebuildExisting bool) []*build.Job {
	var res []*build.Job

	appNameColLen := maxAppNameLen(apps) + sepLen
	inputResolver := baur.NewInputResolver()

	for _, app := range apps {
		task := app.Task()

		log.Debugf("%s: resolving build inputs and calculating digests...", task)
		inputs, err := inputResolver.Resolve(repositoryRootDir, task)
		if err != nil {
			log.Fatalf("%s: resolving input failed: %s\n", task, err)
		}

		digest, err := inputs.Digest()
		if err != nil {
			log.Fatalf("%s: calculating total input digest failed: %s\n", task, err)
		}

		status, existingBuild, err := baur.TaskRunStatusInputs(task, inputs, storer, branchFlag, compareFlag)
		if err != nil {
			log.Fatalf("fetching build from database failed: %s\n", err)
		}

		if status == baur.BuildStatusUndefined {
			panic(fmt.Sprintf("task status is %s and err is not nil (%s) \n", status, err))
		}

		if status == baur.BuildStatusExist {
			fmt.Printf("%-*s%s%s (%s)\n",
				appNameColLen, app.Name, appColSep, coloredBuildStatus(status), highlight(existingBuild.ID))
		} else {
			fmt.Printf("%-*s%s%s\n",
				appNameColLen, app.Name, appColSep, coloredBuildStatus(status))
		}

		if status == baur.BuildStatusInputsUndefined {
			continue
		}

		if !rebuildExisting && status == baur.BuildStatusExist {
			continue
		}

		storageInputs, err := filesToStorageInputs(inputs)
		if err != nil {
			log.Fatalf("%s: %s\n", task, err)
		}

		res = append(res, &build.Job{
			Application: app.Name,
			Directory:   app.Path,
			Command:     app.Task().Command,
			UserData: &buildUserData{
				App:              app,
				Inputs:           storageInputs,
				TotalInputDigest: digest.String(),
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
	store = MustGetPostgresClt(repo)

	apps = mustArgToApps(repo, args)
	baur.SortAppsByName(apps)

	startTs := time.Now()

	fmt.Printf("Evaluating build status of applications:\n")
	buildJobs = pendingBuilds(store, apps, repo.Path, buildForce)

	if buildForce {
		fmt.Printf("\nBuilding applications with build status: %s or %s\n",
			coloredBuildStatus(baur.BuildStatusPending), coloredBuildStatus(baur.BuildStatusExist))
	} else {
		fmt.Printf("\nBuilding applications with build status: %s\n",
			coloredBuildStatus(baur.BuildStatusPending))
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
