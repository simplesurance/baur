package baur

import (
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/xid"

	"github.com/simplesurance/baur/cfg"
	"github.com/simplesurance/baur/digest"
	"github.com/simplesurance/baur/digest/sha384"
)

// App represents an application
type App struct {
	RelPath          string
	Path             string
	Name             string
	BuildCmd         string
	Repository       *Repository
	Outputs          []BuildOutput
	BuildInputPaths  []BuildInputPathResolver
	buildInputs      []BuildInput
	totalInputDigest *digest.Digest
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

func (a *App) setInputsFromCfg(r *Repository, cfg *cfg.App) error {
	sliceLen := len(cfg.Build.Input.Files.Paths)

	if len(cfg.Build.Input.GitFiles.Paths) > 0 {
		sliceLen++
	}

	if len(cfg.Build.Input.GolangSources.Paths) > 0 {
		sliceLen++
	}

	a.BuildInputPaths = make([]BuildInputPathResolver, 0, sliceLen)

	for _, p := range cfg.Build.Input.Files.Paths {
		a.BuildInputPaths = append(a.BuildInputPaths, NewFileGlobPath(r.Path, a.RelPath, p))
	}

	if len(cfg.Build.Input.GitFiles.Paths) > 0 {
		a.BuildInputPaths = append(a.BuildInputPaths,
			NewGitPaths(r.Path, a.RelPath, cfg.Build.Input.GitFiles.Paths))
	}

	if len(cfg.Build.Input.GolangSources.Paths) > 0 {
		var gopath string

		if len(cfg.Build.Input.GolangSources.GoPath) > 0 {
			gopath = filepath.Join(a.Path, cfg.Build.Input.GolangSources.GoPath)
		}

		a.BuildInputPaths = append(a.BuildInputPaths,
			NewGoSrcDirs(r.Path, a.RelPath, gopath, cfg.Build.Input.GolangSources.Paths))
	}

	return nil
}

func (a *App) setDockerOutputsFromCfg(cfg *cfg.App) error {
	for _, di := range cfg.Build.Output.DockerImage {
		tag, err := replaceGitCommitVar(di.RegistryUpload.Tag, a.Repository)
		if err != nil {
			return errors.Wrap(err, "replacing $GITCOMMIT in tag failed")
		}

		tag = replaceUUIDvar(tag)

		a.Outputs = append(a.Outputs, &DockerArtifact{
			ImageIDFile: path.Join(a.Path, di.IDFile),
			Tag:         tag,
			Repository:  di.RegistryUpload.Repository,
		})
	}

	return nil
}

func (a *App) setFileOutputsFromCFG(cfg *cfg.App) error {
	for _, f := range cfg.Build.Output.File {
		destFile, err := replaceGitCommitVar(f.S3Upload.DestFile, a.Repository)
		if err != nil {
			return errors.Wrap(err, "replacing $GITCOMMIT in dest_file failed")
		}
		destFile = replaceUUIDvar(replaceAppNameVar(destFile, a.Name))

		url := "s3://" + f.S3Upload.Bucket + "/" + destFile

		a.Outputs = append(a.Outputs, &FileArtifact{
			RelPath:   path.Join(a.RelPath, f.Path),
			Path:      path.Join(a.Path, f.Path),
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

	appAbsPath := path.Dir(cfgPath)
	appRelPath, err := filepath.Rel(repository.Path, appAbsPath)
	if err != nil {
		return nil, errors.Wrap(err, "resolving repository relative application path failed")
	}

	app := App{
		Repository: repository,
		Path:       path.Dir(cfgPath),
		RelPath:    appRelPath,
		Name:       cfg.Name,
		BuildCmd:   cfg.Build.Command,
	}

	if err := app.setDockerOutputsFromCfg(cfg); err != nil {
		return nil, errors.Wrap(err, "processing docker output declarations failed")
	}

	if err := app.setFileOutputsFromCFG(cfg); err != nil {
		return nil, errors.Wrap(err, "processing S3 output declarations failed")
	}

	if err := app.setInputsFromCfg(repository, cfg); err != nil {
		return nil, errors.Wrap(err, "processing input declarations failed")
	}

	return &app, nil
}

// String returns the string representation of an app
func (a *App) String() string {
	return a.Name
}

// BuildInputs returns all deduplicated BuildInputs.
// If the function is called the first time, the BuildInputPaths are resolved
// and stored. On following calls the stored BuildInputs are returned.
func (a *App) BuildInputs() ([]BuildInput, error) {
	if a.buildInputs != nil {
		return a.buildInputs, nil
	}

	if len(a.BuildInputPaths) == 0 {
		a.buildInputs = []BuildInput{}
		return a.buildInputs, nil
	}

	dedupBuildInputs := map[string]BuildInput{}

	for _, inputPath := range a.BuildInputPaths {
		buildInputs, err := inputPath.Resolve()
		if err != nil {
			return nil, errors.Wrapf(err, "resolving %q failed", inputPath)
		}

		for _, bi := range buildInputs {
			if _, exist := dedupBuildInputs[bi.URI()]; exist {
				continue
			}

			dedupBuildInputs[bi.URI()] = bi
		}
	}

	for _, bi := range dedupBuildInputs {
		a.buildInputs = append(a.buildInputs, bi)
	}

	return a.buildInputs, nil
}

// TotalInputDigest returns the total input digest that is calculated over all
// input sources. The calculation is only done on the 1. call on following calls
// the stored digest is returned
func (a *App) TotalInputDigest() (digest.Digest, error) {
	if a.totalInputDigest != nil {
		return *a.totalInputDigest, nil
	}

	buildInputs, err := a.BuildInputs()
	if err != nil {
		return digest.Digest{}, err
	}

	digests := make([]*digest.Digest, 0, len(buildInputs))
	for _, bi := range buildInputs {
		d, err := bi.Digest()
		if err != nil {
			return digest.Digest{}, errors.Wrapf(err, "calculating input digest of %q failed", bi)
		}

		digests = append(digests, &d)
	}

	totalDigest, err := sha384.Sum(digests)
	if err != nil {
		return digest.Digest{}, errors.Wrap(err, "calculating total input digest")
	}

	a.totalInputDigest = totalDigest

	return *a.totalInputDigest, nil
}

// SortAppsByName sorts the apps in the slice by Name
func SortAppsByName(apps []*App) {
	sort.Slice(apps, func(i int, j int) bool {
		return apps[i].Name < apps[j].Name
	})
}
