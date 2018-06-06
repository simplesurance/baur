package baur

import (
	"path"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/xid"

	"github.com/simplesurance/baur/cfg"
)

// App represents an application
type App struct {
	Dir        string
	Name       string
	BuildCmd   string
	Repository *Repository
	Artifacts  []Artifact
}

func replaceUUIDvar(in string) string {
	return strings.Replace(in, "$UUID", xid.New().String(), -1)
}

func replaceAppNameVar(in, appName string) string {
	return strings.Replace(in, "$APPNAME", appName, -1)
}

func dockerArtifactsFromCFG(appDir, appName string, cfg *cfg.App) []Artifact {
	res := make([]Artifact, 0, len(cfg.DockerArtifact))

	for _, ar := range cfg.DockerArtifact {
		tag := replaceUUIDvar(ar.Tag)

		res = append(res, &DockerArtifact{
			ImageIDFile: path.Join(appDir, ar.IDFile),
			Tag:         tag,
			Repository:  ar.Repository,
		})
	}

	return res
}

func s3ArtifactsFromCFG(appDir, appName string, cfg *cfg.App) []Artifact {
	res := make([]Artifact, 0, len(cfg.S3Artifact))

	for _, ar := range cfg.S3Artifact {
		destFile := replaceUUIDvar(replaceAppNameVar(ar.DestFile, appName))

		url := "s3://" + ar.Bucket + "/" + destFile

		res = append(res, &FileArtifact{
			Path:      path.Join(appDir, ar.Path),
			DestFile:  destFile,
			UploadURL: url,
		})
	}

	return res
}

// NewApp reads the configuration file and returns a new App
func NewApp(repository *Repository, cfgPath string) (*App, error) {
	cfg, err := cfg.AppFromFile(cfgPath)
	if err != nil {
		return nil, errors.Wrapf(err,
			"reading application config %s failed", cfgPath)
	}

	err = cfg.Validate()
	if err != nil {
		return nil, errors.Wrapf(err,
			"validating application config %s failed",
			cfgPath)
	}

	app := App{
		Repository: repository,
		Dir:        path.Dir(cfgPath),
		Name:       cfg.Name,
		BuildCmd:   cfg.BuildCommand,
	}

	if len(app.BuildCmd) == 0 {
		app.BuildCmd = repository.DefaultBuildCmd
	}

	app.Artifacts = append(dockerArtifactsFromCFG(app.Dir, app.Name, cfg),
		s3ArtifactsFromCFG(app.Dir, app.Name, cfg)...)

	return &app, nil
}

// SortAppsByName sorts the apps in the slice by Name
func SortAppsByName(apps []*App) {
	sort.Slice(apps, func(i int, j int) bool {
		return apps[i].Name < apps[j].Name
	})
}

func (a *App) String() string {
	return a.Name
}
