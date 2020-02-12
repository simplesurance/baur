package baur

import (
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pkg/errors"

	"github.com/simplesurance/baur/cfg"
	"github.com/simplesurance/baur/cfg/resolver"
	"github.com/simplesurance/baur/digest"
	"github.com/simplesurance/baur/digest/sha384"
	"github.com/simplesurance/baur/log"
	"github.com/simplesurance/baur/resolve/gitpath"
	"github.com/simplesurance/baur/resolve/glob"
	"github.com/simplesurance/baur/resolve/gosource"
	"github.com/simplesurance/baur/upload/scheduler"
)

// App represents an application
type App struct {
	RelPath          string
	Path             string
	Name             string
	BuildCmd         string
	Outputs          []BuildOutput
	totalInputDigest *digest.Digest

	UnresolvedInputs []*cfg.Input
	buildInputs      []*File

	repositoryRootPath string
}

func (a *App) addBuildOutput(buildOutput *cfg.Output) error {
	if err := a.addDockerBuildOutputs(buildOutput); err != nil {
		return errors.Wrap(err, "error in DockerImage section")
	}

	if err := a.addFileOutputs(buildOutput); err != nil {
		return errors.Wrap(err, "error in File section")
	}

	return nil
}

func (a *App) addDockerBuildOutputs(buildOutput *cfg.Output) error {
	for _, di := range buildOutput.DockerImage {
		a.Outputs = append(a.Outputs, &DockerArtifact{
			ImageIDFile: path.Join(a.Path, di.IDFile),
			Tag:         di.RegistryUpload.Tag,
			Repository:  di.RegistryUpload.Repository,
			Registry:    di.RegistryUpload.Registry,
		})
	}

	return nil
}

func (a *App) addFileOutputs(buildOutput *cfg.Output) error {
	for _, f := range buildOutput.File {
		filePath := f.Path
		if !f.S3Upload.IsEmpty() {
			url := "s3://" + f.S3Upload.Bucket + "/" + f.S3Upload.DestFile

			src := path.Join(a.Path, filePath)

			a.Outputs = append(a.Outputs, &FileArtifact{
				RelPath:   path.Join(a.RelPath, filePath),
				Path:      src,
				DestFile:  f.S3Upload.DestFile,
				UploadURL: url,
				uploadJob: &scheduler.S3Job{
					DestURL:  url,
					FilePath: src,
				},
			})
		}

		if !f.FileCopy.IsEmpty() {
			src := path.Join(a.Path, filePath)

			a.Outputs = append(a.Outputs, &FileArtifact{
				RelPath:   path.Join(a.RelPath, filePath),
				Path:      src,
				DestFile:  f.FileCopy.Path,
				UploadURL: f.FileCopy.Path,
				uploadJob: &scheduler.FileCopyJob{
					Src: src,
					Dst: f.FileCopy.Path,
				},
			})
		}
	}

	return nil
}

func (a *App) addCfgsToBuildInputs(appCfg *cfg.App) {
	buildInput := cfg.Input{}
	buildInput.Files.Paths = append(buildInput.Files.Paths, AppCfgFile)

	a.UnresolvedInputs = append(a.UnresolvedInputs, &buildInput)
}

// NewApp reads the configuration file and returns a new App
func NewApp(includeDB *cfg.IncludeDB, repositoryRootPath, cfgPath, curGitCommit string) (*App, error) {
	appCfg, err := cfg.AppFromFile(cfgPath)
	if err != nil {
		return nil, errors.Wrapf(err,
			"reading application config %s failed", cfgPath)
	}

	var errAppName string
	if appCfg.Name != "" {
		errAppName = appCfg.Name
	} else {
		errAppName = cfgPath
	}

	err = appCfg.Merge(includeDB, &resolver.StrReplacement{Old: rootVarName, New: repositoryRootPath})
	if err != nil {
		return nil, errors.Wrapf(err,
			"%s: merging includes failed", errAppName)
	}

	resolvers := DefaultAppCfgResolvers(repositoryRootPath, appCfg.Name, curGitCommit)
	err = appCfg.Resolve(resolvers)
	if err != nil {
		return nil, errors.Wrapf(err,
			"%s: resolving variables in config failed", errAppName)
	}

	err = appCfg.Validate()
	if err != nil {
		return nil, errors.Wrapf(err,
			"validating application config %s failed", cfgPath)
	}

	appAbsPath := path.Dir(cfgPath)
	appRelPath, err := filepath.Rel(repositoryRootPath, appAbsPath)
	if err != nil {
		return nil, errors.Wrapf(err, "%s: resolving repository relative application path failed", appCfg.Name)
	}

	var buildCommand string
	if len(appCfg.Tasks) > 1 {
		return nil, fmt.Errorf("%s: has >1 tasks defined, only 1 task definition with name 'build' is currently allowed", appCfg.Name)
	}

	var buildTask *cfg.Task
	if len(appCfg.Tasks) == 1 {
		buildTask = appCfg.Tasks[0]

		if buildTask.Name != "build" {
			return nil, fmt.Errorf("%s: has a task defined with name %q, only 1 task definition with name 'build' is currently allowed", appCfg.Name, buildTask.Name)
		}

		buildCommand = buildTask.Command
	}

	app := App{
		Path:               path.Dir(cfgPath),
		RelPath:            appRelPath,
		Name:               appCfg.Name,
		BuildCmd:           strings.TrimSpace(buildCommand),
		repositoryRootPath: repositoryRootPath,
	}

	if buildTask == nil {
		return &app, nil
	}

	err = app.addBuildOutput(&buildTask.Output)
	if err != nil {
		return nil, errors.Wrapf(err, "%s: processing Task.Output section failed", app.Name)
	}

	app.UnresolvedInputs = []*cfg.Input{&buildTask.Input}
	app.addCfgsToBuildInputs(appCfg)

	return &app, nil
}

// String returns the string representation of an app
func (a *App) String() string {
	return a.Name
}

func (a *App) pathsToUniqFiles(workingDir string, paths []string) ([]*File, error) {
	dedupMap := make(map[string]struct{}, len(paths))
	res := make([]*File, 0, len(paths))

	for _, path := range paths {
		if _, exist := dedupMap[path]; exist {
			log.Debugf("%s: removed duplicate Build Input '%s'", a.Name, path)
			continue
		}
		dedupMap[path] = struct{}{}

		relPath, err := filepath.Rel(workingDir, path)
		if err != nil {
			return nil, err
		}

		// TODO: should resolving the relative path be done in
		// Newfile() instead?
		res = append(res, NewFile(workingDir, relPath))
	}

	return res, nil
}

func (a *App) resolveGlobFileInputs() ([]string, error) {
	var res []string

	for _, bi := range a.UnresolvedInputs {
		for _, globPath := range bi.Files.Paths {
			if !filepath.IsAbs(globPath) {
				globPath = filepath.Join(a.Path, globPath)
			}

			resolver := glob.NewResolver(globPath)
			paths, err := resolver.Resolve()
			if err != nil {
				return nil, errors.Wrap(err, globPath)
			}

			if len(paths) == 0 {
				return nil, fmt.Errorf("'%s' matched 0 files", globPath)
			}

			res = append(res, paths...)
		}
	}

	return res, nil
}

func (a *App) resolveGitFileInputs() ([]string, error) {
	var res []string

	for _, bi := range a.UnresolvedInputs {
		if len(bi.GitFiles.Paths) == 0 {
			continue
		}

		resolver := gitpath.NewResolver(a.Path, bi.GitFiles.Paths...)
		paths, err := resolver.Resolve()
		if err != nil {
			return nil, err
		}

		if len(paths) == 0 {
			return nil, fmt.Errorf("'%s' matched 0 files", strings.Join(paths, ", "))
		}
		res = append(res, paths...)
	}

	return res, nil
}

func (a *App) resolveGoSrcInputs() ([]string, error) {
	var res []string

	for _, bi := range a.UnresolvedInputs {
		if len(bi.GolangSources.Paths) == 0 {
			continue
		}

		absGoSourcePaths := make([]string, 0, len(bi.GolangSources.Paths))
		for _, relGosrcpath := range bi.GolangSources.Paths {
			absPath := path.Join(a.Path, relGosrcpath)
			absGoSourcePaths = append(absGoSourcePaths, absPath)
		}

		resolver := gosource.NewResolver(log.Debugf, bi.GolangSourcesInputs().Environment, absGoSourcePaths...)
		paths, err := resolver.Resolve()
		if err != nil {
			return nil, err
		}

		if len(paths) == 0 {
			return nil, fmt.Errorf("'%s' matched 0 files", strings.Join(paths, ", "))
		}

		res = append(res, paths...)
	}

	return res, nil
}

func (a *App) resolveBuildInputPaths() ([]string, error) {
	globPaths, err := a.resolveGlobFileInputs()
	if err != nil {
		return nil, errors.Wrapf(err, "resolving File BuildInputs failed")
	}

	gitPaths, err := a.resolveGitFileInputs()
	if err != nil {
		return nil, errors.Wrapf(err, "resolving GitFile BuildInputs failed")
	}

	goSrcPaths, err := a.resolveGoSrcInputs()
	if err != nil {
		return nil, errors.Wrapf(err, "resolving GoLangSources BuildInputs failed")
	}

	paths := make([]string, 0, len(globPaths)+len(gitPaths)+len(goSrcPaths))
	paths = append(paths, globPaths...)
	paths = append(paths, gitPaths...)
	paths = append(paths, goSrcPaths...)

	return paths, nil
}

// HasBuildInputs returns true if BuildInputs are defined for the app
func (a *App) HasBuildInputs() bool {
	for _, bi := range a.UnresolvedInputs {
		if len(bi.Files.Paths) != 0 {
			return true
		}

		if len(bi.GitFiles.Paths) != 0 {
			return true
		}

		if len(bi.GolangSources.Paths) != 0 {
			return true
		}
	}

	return false
}

// BuildInputs resolves all build inputs of the app.
// The BuildInputs are deduplicates before they are returned.
// If one more resolved path does not match a file an error is generated.
// If not build inputs are defined, an empty slice and no error is returned.
// If the function is called the first time, the BuildInputPaths are resolved
// and stored. On following calls the stored BuildInputs are returned.
func (a *App) BuildInputs() ([]*File, error) {
	if a.buildInputs != nil {
		return a.buildInputs, nil
	}

	paths, err := a.resolveBuildInputPaths()
	if err != nil {
		return nil, err
	}

	a.buildInputs, err = a.pathsToUniqFiles(a.repositoryRootPath, paths)
	if err != nil {
		return nil, err
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
