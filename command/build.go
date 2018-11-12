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
)

var buildLongHelp = fmt.Sprintf(`
Build applications.
If no path or application name is passed, all applications in the repository are build.
By default only applications with status '%s' and '%s' are build.

Environment Variables:
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
	highlight("Outstanding"),
	highlight("Input Undefined"),

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
build payment-service     build the application with the name payment-service
build ---force            rebuild all applications in the repository
build --verbose ui/shop   build the application in the directory ui/shop with verbose output
build --upload ui/shop    build the application in the directory ui/shop and upload it's outputs`

var buildCmd = &cobra.Command{
	Use:     "build [<PATH>|<APP-NAME>]...",
	Short:   "build applications",
	Long:    strings.TrimSpace(buildLongHelp),
	Run:     buildRun,
	Example: strings.TrimSpace(buildExampleHelp),
	Args:    cobra.ArbitraryArgs,
}

var (
	buildUpload bool
	buildForce  bool

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
	buildCmd.Flags().BoolVar(&buildUpload, "upload", false,
		"upload build outputs after the application(s) was build")
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

	resultLock.Lock()
	defer resultLock.Unlock()

	b, exist := result[appName]
	if !exist {
		log.Fatalf("resultAddUploadResult: %q does not exist in build result map", appName)
	}

	switch r.Job.Type() {
	case scheduler.JobDocker:
		arType = storage.DockerArtifact
	case scheduler.JobFileCopy:
		fallthrough
	case scheduler.JobS3:
		arType = storage.FileArtifact
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
			Directory: app.Path,
			Command:   app.BuildCmd,
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

func appsToString(apps []*baur.App) string {
	var res string

	for i, a := range apps {
		res += a.Name

		if i < len(apps)-1 {
			res += ", "

			if (i+1)%5 == 0 {
				res += "\n"
			}
		}
	}

	return res
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

		fmt.Printf("%s: %s uploaded to %s (%.3fs)\n",
			ud.App.Name, ud.Output.LocalPath(), res.URL, res.Duration.Seconds())

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

func outstandingBuilds(storage storage.Storer, apps []*baur.App) []*baur.App {
	var res []*baur.App

	maxAppNameLen := maxAppNameLen(apps) + 4

	for _, app := range apps {
		buildStatus, _, _ := mustGetBuildStatus(app, storage)

		if buildStatus != baur.BuildStatusExist {
			res = append(res, app)
		}

		fmt.Printf("%-*s => %s\n", maxAppNameLen, app.Name, coloredBuildStatus((buildStatus)))
	}

	fmt.Println()

	return res
}

func buildRun(cmd *cobra.Command, args []string) {
	var apps []*baur.App
	var uploadWatchFin chan struct{}
	var uploader scheduler.Manager

	repo := MustFindRepository()
	startTs := time.Now()

	apps = mustArgToApps(repo, args)

	if buildUpload || !buildForce {
		store = MustGetPostgresClt(repo)
	}

	if !buildForce {
		fmt.Printf("Identifying applications with outstanding builds:\n\n")
		apps = outstandingBuilds(store, apps)
	}

	if len(apps) == 0 {
		fmt.Println()
		term.PrintSep()
		fmt.Println("Application build(s) already exist, nothing to build, see 'baur ls -b'.\n" +
			"If you want to rebuild applications pass '-f' to 'baur build'")
		os.Exit(0)
	}

	baur.SortAppsByName(apps)

	repo.GitCommitID()

	buildJobs := createBuildJobs(apps)
	buildChan := make(chan *build.Result, len(apps))
	builder := seq.New(buildJobs, buildChan)
	outputCnt := outputCount(apps)

	if buildUpload {
		uploadChan := make(chan *scheduler.Result, outputCnt)
		uploader = startBGUploader(outputCnt, uploadChan)
		uploadWatchFin = make(chan struct{}, 1)
		go waitPrintUploadStatus(uploader, uploadChan, uploadWatchFin, outputCnt)

		fmt.Printf("building and uploading the following applications:\n%s\n",
			appsToString(apps))
	} else {
		fmt.Printf("building the following applications:\n%s\n",
			appsToString(apps))
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

			if buildUpload {
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

	if buildUpload && outputCnt > 0 {
		fmt.Println("waiting for uploads to finish...")
		<-uploadWatchFin
	}

	term.PrintSep()
	fmt.Printf("finished in %s\n", time.Since(startTs))
}

func mustGetBuildStatus(app *baur.App, storage storage.Storer) (baur.BuildStatus, *storage.BuildWithDuration, string) {
	var strBuildID string

	status, build, err := baur.GetBuildStatus(storage, app)
	if err != nil {
		log.Fatalf("%s: evaluating build status of failed: %s", app, err)
	}

	if build != nil {
		strBuildID = fmt.Sprint(build.ID)
	}

	return status, build, strBuildID
}
