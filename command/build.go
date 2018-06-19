package command

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/simplesurance/baur"
	"github.com/simplesurance/baur/build"
	"github.com/simplesurance/baur/build/seq"
	"github.com/simplesurance/baur/docker"
	"github.com/simplesurance/baur/log"
	"github.com/simplesurance/baur/prettyprint"
	"github.com/simplesurance/baur/s3"
	"github.com/simplesurance/baur/storage"
	"github.com/simplesurance/baur/storage/postgres"
	"github.com/simplesurance/baur/term"
	"github.com/simplesurance/baur/upload"
	sequploader "github.com/simplesurance/baur/upload/seq"
	"github.com/spf13/cobra"
)

var buildUpload bool

type uploadUserData struct {
	App      *baur.App
	Artifact baur.Artifact
}

var result = map[string]*storage.Build{}
var resultLock = sync.Mutex{}

var store storage.Storer

func resultAddBuildResult(app *baur.App, r *build.Result) {
	resultLock.Lock()
	defer resultLock.Unlock()

	b := storage.Build{
		AppName:        app.Name,
		StartTimeStamp: r.StartTs,
		StopTimeStamp:  r.StopTs,
		// Sources: // TODO
		//TotalSrcHash: // TODO
	}

	result[app.Name] = &b

}

func resultAddUploadResult(appName string, ar baur.Artifact, r *upload.Result) {
	var arType storage.ArtifactType

	resultLock.Lock()
	defer resultLock.Unlock()

	b, exist := result[appName]
	if !exist {
		panic(fmt.Sprintf("resultAddUploadResult: %q does not exist in build result map", appName))
	}

	if r.Job.Type() == upload.JobDocker {
		arType = storage.DockerArtifact
	} else if r.Job.Type() == upload.JobS3 {
		arType = storage.S3Artifact
	}

	b.Artifacts = append(b.Artifacts, &storage.Artifact{
		Name: ar.Name(),
		// SizeBytes, // TODO implement it
		Type:           arType,
		URL:            r.URL,
		UploadDuration: r.Duration,
	})
}

func recordResultIsComplete(app *baur.App) (bool, *storage.Build) {
	resultLock.Lock()
	defer resultLock.Unlock()

	b, exist := result[app.Name]
	if !exist {
		panic(fmt.Sprintf("recordResultIfComplete: %q does not exist in build result map", app.Name))
	}

	if len(app.Artifacts) == len(b.Artifacts) {
		return true, b
	}

	return false, nil

}

func init() {
	buildCmd.Flags().BoolVar(&buildUpload, "upload", false,
		"upload build artifacts after the application(s) was build")
	rootCmd.AddCommand(buildCmd)
}

const buildLongHelp = `
Builds applications.
If no argument is the application in the current directory is build.
If the current directory does not contain an application, all applications are build.

Environment Variables:
The following environment variables configure credentials for artifact repositories:

  S3 Repositories:
    AWS_REGION
    AWS_ACCESS_KEY_ID
    AWS_SECRET_ACCESS_KEY

  Docker Repositories:
    DOCKER_USERNAME
    DOCKER_PASSWORD
`

const buildExampleHelp = `
baur build 		       build the applications in the current directory
baur build all		       build all applications in the repository
baur build payment-service     build the application with the name payment-service
baur build --verbose ui/shop   build the application in the directory ui/shop with verbose output
baur build --upload ui/shop    build the application in the directory ui/shop and upload it's artifacts`

var buildCmd = &cobra.Command{
	Use:     "build [<PATH>|<APP-NAME>|all]",
	Short:   "builds an application",
	Long:    strings.TrimSpace(buildLongHelp),
	Run:     buildCMD,
	Example: strings.TrimSpace(buildExampleHelp),
	Args:    cobra.MaximumNArgs(1),
}

func mustArgToApps(repo *baur.Repository, arg string) []*baur.App {
	if strings.ToLower(arg) == "all" {
		apps, err := repo.FindApps()
		if err != nil {
			log.Fatalln(err)
		}

		return apps
	}

	return []*baur.App{mustArgToApp(repo, arg)}
}

func artifactCount(apps []*baur.App) int {
	var cnt int

	for _, a := range apps {
		cnt += len(a.Artifacts)
	}

	return cnt
}

func dockerAuthFromEnv() (string, string) {
	return os.Getenv("DOCKER_USERNAME"), os.Getenv("DOCKER_PASSWORD")
}

func createBuildJobs(apps []*baur.App) []*build.Job {
	buildJobs := make([]*build.Job, 0, len(apps))

	for _, app := range apps {
		buildJobs = append(buildJobs, &build.Job{
			Directory: app.Dir,
			Command:   app.BuildCmd,
			UserData:  app,
		})
	}

	return buildJobs

}

func startBGUploader(artifactCnt int, uploadChan chan *upload.Result) upload.Manager {
	s3Uploader, err := s3.NewClient()
	if err != nil {
		log.Fatalln(err.Error)
	}

	dockerUser, dockerPass := dockerAuthFromEnv()
	dockerUploader, err := docker.NewClient(dockerUser, dockerPass)
	if err != nil {
		log.Fatalln(err.Error)
	}

	uploader := sequploader.New(s3Uploader, dockerUploader, uploadChan)

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

func waitPrintUploadStatus(uploader upload.Manager, uploadChan chan *upload.Result, finished chan struct{}, artifactCnt int) {
	var resultCnt int

	for res := range uploadChan {
		ud, ok := res.Job.GetUserData().(*uploadUserData)
		if !ok {
			panic("upload result user data has unexpected type")
		}

		if res.Err != nil {
			log.Fatalf("upload of %q failed: %s\n", ud.Artifact, res.Err)
		}

		log.Actionf("%s: artifact %s uploaded to %s (%.3fs)\n",
			ud.App.Name, ud.Artifact.LocalPath(), res.URL, res.Duration.Seconds())

		resultAddUploadResult(ud.App.Name, ud.Artifact, res)

		complete, build := recordResultIsComplete(ud.App)
		if complete {
			if err := store.Save(build); err != nil {
				log.Fatalf("storing build information about %q failed: %s", ud.App.Name, err)
			}

			log.Debugf("stored the following build information: %s\n", prettyprint.AsString(build))
		}

		resultCnt++
		if resultCnt == artifactCnt {
			break
		}
	}

	uploader.Stop()

	close(finished)
}

func buildCMD(cmd *cobra.Command, args []string) {
	var apps []*baur.App
	var uploadWatchFin chan struct{}
	var uploader upload.Manager

	repo := mustFindRepository()
	startTs := time.Now()

	if len(args) > 0 {
		apps = mustArgToApps(repo, args[0])
	} else if isAppDir(".") {
		apps = mustArgToApps(repo, ".")
	} else {
		apps = mustArgToApps(repo, "all")
	}

	baur.SortAppsByName(apps)

	buildJobs := createBuildJobs(apps)
	buildChan := make(chan *build.Result, len(apps))
	builder := seq.New(buildJobs, buildChan)
	artifactCnt := artifactCount(apps)

	if buildUpload {
		var err error
		store, err = postgres.New(repo.PSQLURL)
		if err != nil {
			log.Fatalf("could not establish connection to postgreSQL db: %s", err)
		}

		uploadChan := make(chan *upload.Result, artifactCnt)
		uploader = startBGUploader(artifactCnt, uploadChan)
		uploadWatchFin = make(chan struct{}, 1)
		go waitPrintUploadStatus(uploader, uploadChan, uploadWatchFin, artifactCnt)

		log.Actionf("building and uploading the following applications: \n%s\n",
			appsToString(apps))
	} else {
		log.Actionf("building the following applications: \n%s\n",
			appsToString(apps))
	}

	term.PrintSep()

	go builder.Start()

	for status := range buildChan {
		app := status.Job.UserData.(*baur.App)

		if status.Error != nil {
			log.Fatalf("%s: build failed: %s\n", app.Name, status.Error)
		}

		if status.ExitCode != 0 {
			log.Fatalf("%s: build failed: command (%q) exited with code %d "+
				"Output: %s\n",
				app.Name, status.Job.Command, status.ExitCode, status.Output)
		}

		log.Actionf("%s: build successful (%.3fs)\n", app.Name, status.StopTs.Sub(status.StartTs).Seconds())
		resultAddBuildResult(app, status)

		for _, ar := range app.Artifacts {
			if !ar.Exists() {
				log.Fatalf("artifact %q of %s did not exist after build\n",
					ar, app)
			}

			if buildUpload {
				uj, err := ar.UploadJob()
				if err != nil {
					log.Fatalf("could not get upload job for artifact %s: %s", ar, err)
				}

				uj.SetUserData(&uploadUserData{
					App:      app,
					Artifact: ar,
				})

				uploader.Add(uj)

			}
			log.Actionf("%s: created artifact %s\n", app.Name, ar)
		}

	}

	if buildUpload && artifactCnt > 0 {
		log.Actionf("waiting for uploads to finish...\n")
		<-uploadWatchFin
	}

	term.PrintSep()
	log.Infof("finished in %s\n", time.Since(startTs))

}
