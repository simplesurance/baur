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
	"github.com/simplesurance/baur/digest"
	"github.com/simplesurance/baur/digest/sha384"
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
Build applications.
If no path or application name is passed, all applications in the repository are build.
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
	App              *baur.App
	Inputs           []*storage.Input
	TotalInputDigest string
}

func init() {
	buildCmd.Flags().BoolVarP(&buildSkipUpload, "skip-upload", "s", false,
		"skip uploading build outputs and recording the build")
	buildCmd.Flags().BoolVarP(&buildForce, "force", "f", false,
		"force rebuilding of all applications")
	rootCmd.AddCommand(buildCmd)
}

func resultAddBuildResult(bud *buildUserData, r *build.Result) {
	resultLock.Lock()
	defer resultLock.Unlock()

	b := storage.Build{
		Application: storage.Application{Name: bud.App.Name},
		VCSState: storage.VCSState{
			CommitID: mustGetCommitID(bud.App.Repository),
			IsDirty:  mustGetGitWorktreeIsDirty(bud.App.Repository),
		},
		StartTimeStamp:   r.StartTs,
		StopTimeStamp:    r.StopTs,
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

func recordResultIsComplete(app *baur.App) (bool, *storage.Build) {
	resultLock.Lock()
	defer resultLock.Unlock()

	b, exist := result[app.Name]
	if !exist {
		log.Fatalf("recordResultIfComplete: %q does not exist in build result map", app.Name)
	}

	if len(app.Outputs) == len(b.Outputs) {
		return true, b
	}

	return false, nil

}

func outputCount(apps []*baur.App) int {
	var cnt int

	for _, a := range apps {
		cnt += len(a.Outputs)
	}

	return cnt
}

func dockerAuthFromEnv() (string, string) {
	return os.Getenv(dockerEnvUsernameVar), os.Getenv(dockerEnvPasswordVar)
}

func calcDigests(app *baur.App) ([]*storage.Input, string) {
	var totalDigest string
	var storageInputs []*storage.Input
	inputDigests := []*digest.Digest{}

	// TODO: refactor this functions, most is obsolete and can be replaced
	// by App.TotalInputDigests()
	// The storageInputs can be removed, apps.BuildInputs() can be used
	// instead to later fill the struct for the db

	log.Debugf("%s: resolving build inputs and calculating digests...", app)
	buildInputs, err := app.BuildInputs()
	if err != nil {
		log.Fatalf("%s: resolving build input paths failed: %s\n", app, err)
	}

	for _, s := range buildInputs {
		d, err := s.Digest()
		if err != nil {
			log.Fatalf("%s: calculating build input digest failed: %s", app, err)
		}

		storageInputs = append(storageInputs, &storage.Input{
			Digest: d.String(),
			URI:    s.RepoRelPath(),
		})

		inputDigests = append(inputDigests, &d)
	}

	if len(inputDigests) > 0 {
		td, err := sha384.Sum(inputDigests)
		if err != nil {
			log.Fatalf("%s: calculating total input digest failed: %s", app, err)
		}

		totalDigest = td.String()
	}

	return storageInputs, totalDigest
}

func createBuildJobs(apps []*baur.App) []*build.Job {
	buildJobs := make([]*build.Job, 0, len(apps))

	for _, app := range apps {
		buildInputs, totalDigest := calcDigests(app)
		log.Debugf("%s: total input digest: %s\n", app, totalDigest)

		buildJobs = append(buildJobs, &build.Job{
			Application: app.Name,
			Directory:   app.Path,
			Command:     app.BuildCmd,
			UserData: &buildUserData{
				App:              app,
				Inputs:           buildInputs,
				TotalInputDigest: totalDigest,
			},
		})
	}

	return buildJobs
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

		complete, build := recordResultIsComplete(ud.App)
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

func appsWithBuildCommand(apps []*baur.App) []*baur.App {
	res := make([]*baur.App, 0, len(apps))

	appNameColLen := maxAppNameLen(apps) + sepLen

	for _, app := range apps {
		if len(app.BuildCmd) == 0 {
			fmt.Printf("%-*s%s%s\n",
				appNameColLen, app.Name, appColSep, coloredBuildStatus(baur.BuildStatusBuildCommandUndefined))
			continue
		}

		fmt.Printf("%-*s%s%s\n",
			appNameColLen, app.Name, appColSep, coloredBuildStatus(baur.BuildStatusPending))
		res = append(res, app)
	}

	return res
}

func pendingBuilds(storage storage.Storer, apps []*baur.App) []*baur.App {
	var res []*baur.App

	appNameColLen := maxAppNameLen(apps) + sepLen

	for _, app := range apps {
		buildStatus, build, _ := mustGetBuildStatus(app, storage)

		if buildStatus == baur.BuildStatusExist {
			fmt.Printf("%-*s%s%s (%s)\n",
				appNameColLen, app.Name, appColSep, coloredBuildStatus(buildStatus), highlight(build.ID))
			continue
		}

		fmt.Printf("%-*s%s%s\n",
			appNameColLen, app.Name, appColSep, coloredBuildStatus(buildStatus))

		if buildStatus == baur.BuildStatusBuildCommandUndefined {
			continue
		}

		res = append(res, app)

	}

	return res
}

func buildRun(cmd *cobra.Command, args []string) {
	var apps []*baur.App
	var uploadWatchFin chan struct{}
	var uploader scheduler.Manager

	repo := MustFindRepository()

	if !buildSkipUpload || !buildForce {
		store = MustGetPostgresClt(repo)
	}

	startTs := time.Now()

	apps = mustArgToApps(repo, args)
	baur.SortAppsByName(apps)

	fmt.Printf("Evaluating build status of applications:\n")
	if buildForce {
		apps = appsWithBuildCommand(apps)
	} else {
		apps = pendingBuilds(store, apps)
	}

	fmt.Println()
	fmt.Printf("Building applications with build status: %s\n",
		coloredBuildStatus(baur.BuildStatusPending))

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

	buildJobs := createBuildJobs(apps)
	buildChan := make(chan *build.Result, len(apps))
	builder := seq.New(buildJobs, buildChan)
	outputCnt := outputCount(apps)

	if !buildSkipUpload {
		uploadChan := make(chan *scheduler.Result, outputCnt)
		uploader = startBGUploader(outputCnt, uploadChan)
		uploadWatchFin = make(chan struct{}, 1)
		go waitPrintUploadStatus(uploader, uploadChan, uploadWatchFin, outputCnt)
	}

	term.PrintSep()

	go builder.Start()

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
		resultAddBuildResult(bud, status)

		for _, ar := range app.Outputs {
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

	term.PrintSep()
	fmt.Printf("finished in %ss\n", durationToStrSeconds(time.Since(startTs)))
}

func mustGetBuildStatus(app *baur.App, storage storage.Storer) (baur.BuildStatus, *storage.BuildWithDuration, string) {
	var strBuildID string

	status, build, err := baur.GetBuildStatus(storage, app)
	if err != nil {
		log.Fatalf("%s: %s", app.Name, err)
	}

	if build != nil {
		strBuildID = fmt.Sprint(build.ID)
	}

	return status, build, strBuildID
}
