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
	Sources    []SrcResolver
}

func replaceUUIDvar(in string) string {
	return strings.Replace(in, "$UUID", xid.New().String(), -1)
}

func replaceAppNameVar(in, appName string) string {
	return strings.Replace(in, "$APPNAME", appName, -1)
}

func replaceGitCommitVar(in string, r *Repository) (string, error) {
	commitID, err := r.GitCommitID()
	if err != nil {
		return "", err
	}

	return strings.Replace(in, "$GITCOMMIT", commitID, -1), nil
}

func (a *App) setSourcesFromCfg(cfg *cfg.App) error {
	sliceLen := len(cfg.SourceFiles.Paths)
	if len(cfg.GitSourceFiles.Paths) > 0 {
		sliceLen++
	}

	a.Sources = make([]SrcResolver, 0, sliceLen)

	for _, p := range cfg.SourceFiles.Paths {
		a.Sources = append(a.Sources, NewFileSrc(a.Dir, p))
	}

	if len(cfg.GitSourceFiles.Paths) > 0 {
		a.Sources = append(a.Sources, NewGitPaths(a.Dir, cfg.GitSourceFiles.Paths))
	}

	return nil
}

func (a *App) setDockerArtifactsFromCfg(cfg *cfg.App) error {
	for _, ar := range cfg.DockerArtifact {
		tag, err := replaceGitCommitVar(ar.Tag, a.Repository)
		if err != nil {
			return errors.Wrap(err, "replacing $GITCOMMIT in tag failed")
		}

		tag = replaceUUIDvar(tag)

		a.Artifacts = append(a.Artifacts, &DockerArtifact{
			ImageIDFile: path.Join(a.Dir, ar.IDFile),
			Tag:         tag,
			Repository:  ar.Repository,
		})
	}

	return nil
}

func (a *App) setS3ArtifactsFromCFG(cfg *cfg.App) error {
	for _, ar := range cfg.S3Artifact {
		destFile, err := replaceGitCommitVar(ar.DestFile, a.Repository)
		if err != nil {
			return errors.Wrap(err, "replacing $GITCOMMIT in dest_file failed")
		}
		destFile = replaceUUIDvar(replaceAppNameVar(destFile, a.Name))

		url := "s3://" + ar.Bucket + "/" + destFile

		a.Artifacts = append(a.Artifacts, &FileArtifact{
			RelPath:   ar.Path,
			Path:      path.Join(a.Dir, ar.Path),
			DestFile:  destFile,
			UploadURL: url,
		})
	}

	return nil
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

	if err := app.setDockerArtifactsFromCfg(cfg); err != nil {
		return nil, errors.Wrap(err, "processing docker artifact declarations failed")
	}

	if err := app.setS3ArtifactsFromCFG(cfg); err != nil {
		return nil, errors.Wrap(err, "processing S3 artifact declarations failed")
	}

	if err := app.setSourcesFromCfg(cfg); err != nil {
		return nil, errors.Wrap(err, "processing Sources declarations failed")
	}

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
